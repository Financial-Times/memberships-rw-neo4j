package main

import (
	"fmt"
	"github.com/Financial-Times/neo-cypher-runner-go"
	"github.com/jmcvetta/neoism"
	"time"
)

type MembershipsDriver interface {
	Write(m membership) error
	Read(uuid string) (m membership, found bool, err error)
	Delete(uuid string) error
}

type MembershipsCypherDriver struct {
	cypherRunner neocypherrunner.CypherRunner
}

func NewMembershipsCypherDriver(cypherRunner neocypherrunner.CypherRunner) MembershipsCypherDriver {
	return MembershipsCypherDriver{cypherRunner}
}

func (mcd MembershipsCypherDriver) Read(uuid string) (membership, bool, error) {
	fmt.Println("HERE")

	results := []struct {
		UUID              string     `json:"uuid"`
		FactsetIdentifier string     `json:"factsetIdentifier"`
		PrefLabel         string     `json:"prefLabel"`
		PersonUUID        string     `json:"personUuid"`
		OrganisationUUID  string     `json:"organisationUuid"`
		InceptionDate     *time.Time `json:"inceptionDate"`
		TerminationDate   *time.Time `json:"terminationDate"`
		MembershipRoles   []role     `json:"membershipRoles"`
	}{}

	query := &neoism.CypherQuery{
		Statement: `MATCH (m:Membership {uuid:{uuid}})-[:HAS_ORGANISATION]->(o:Thing)
					OPTIONAL MATCH (p:Thing)<-[:HAS_MEMBER]-(m)
					OPTIONAL MATCH (r:Thing)<-[rr:HAS_ROLE]-(m)
					WITH p, m, o, collect({roleuuid:r.uuid,inceptionDate:rr.inceptionDate,terminationDate:rr.terminationDate }) as membershipRoles
					return m.uuid as uuid ,m.prefLabel as prefLabel,m.factsetIdentifier as factsetIdentifier,m.inceptionDate as inceptionDate, 
					m.terminationDate as terminationDate, o.uuid as organisationUuid, p.uuid as personUuid,membershipRoles`,

		Parameters: map[string]interface{}{
			"uuid": uuid,
		},
		Result: &results,
	}
	err := mcd.cypherRunner.CypherBatch([]*neoism.CypherQuery{query})

	if err != nil {
		return membership{}, false, err
	}

	if len(results) == 0 {
		return membership{}, false, nil
	}

	result := results[0]

	m := membership{
		UUID:             result.UUID,
		PrefLabel:        result.PrefLabel,
		PersonUUID:       result.PersonUUID,
		OrganisationUUID: result.OrganisationUUID,
		InceptionDate:    result.InceptionDate,
		TerminationDate:  result.TerminationDate,
		MembershipRoles:  result.MembershipRoles,
	}

	if result.FactsetIdentifier != "" {
		m.Identifiers = append(m.Identifiers, identifier{fsAuthority, result.FactsetIdentifier})
	}
	return m, true, nil
}

func (mcd MembershipsCypherDriver) Write(m membership) error {

	params := map[string]interface{}{
		"uuid": m.UUID,
	}

	if m.PrefLabel != "" {
		params["prefLabel"] = m.PrefLabel
	}

	if m.InceptionDate != nil {
		params["inceptionDate"] = m.InceptionDate
	}

	if m.TerminationDate != nil {
		params["terminationDate"] = m.TerminationDate
	}

	for _, identifier := range m.Identifiers {
		if identifier.Authority == fsAuthority {
			params["factsetIdentifier"] = identifier.IdentifierValue
		}
	}
	query := &neoism.CypherQuery{
		Statement: `MERGE (m:Thing {uuid: {uuid}}) 
					MERGE (p:Thing {uuid: {personuuid}})
					MERGE (o:Thing {uuid: {organisationuuid}})
					MERGE (m)-[:HAS_MEMBER]->(p)
		            MERGE (m)-[:HAS_ORGANISATION]->(o)
					set m={allprops}
					set m :Concept
					set m :Membership
		`,
		Parameters: map[string]interface{}{
			"uuid":             m.UUID,
			"allprops":         params,
			"personuuid":       m.PersonUUID,
			"organisationuuid": m.OrganisationUUID,
		},
	}

	queries := []*neoism.CypherQuery{query}

	queryDelRolesRel := &neoism.CypherQuery{
		Statement: `MATCH (m:Thing {uuid: {uuid}})
					OPTIONAL MATCH (r:Thing)<-[rr:HAS_ROLE]-(m)
					DELETE  rr
		`,
		Parameters: map[string]interface{}{
			"uuid": m.UUID,
		},
	}
	queries = append(queries, queryDelRolesRel)

	for _, mr := range m.MembershipRoles {
		rrparams := make(map[string]interface{})

		if mr.InceptionDate != nil {
			rrparams["inceptionDate"] = mr.InceptionDate
		}
		if mr.TerminationDate != nil {
			rrparams["terminationDate"] = mr.TerminationDate
		}

		query := &neoism.CypherQuery{
			Statement: `
				MERGE (m:Thing {uuid:{muuid}})
				MERGE (r:Thing {uuid:{ruuid}})
				CREATE (m)-[rel:HAS_ROLE]->(r)
				SET rel={rrparams}
			`,
			Parameters: map[string]interface{}{
				"muuid":    m.UUID,
				"ruuid":    mr.RoleUUID,
				"rrparams": rrparams,
			},
		}

		queries = append(queries, query)
	}
	return mcd.cypherRunner.CypherBatch(queries)
}

func (mcd MembershipsCypherDriver) Delete(uuid string) error {

	clearNode := &neoism.CypherQuery{
		Statement: `
				MATCH (m:Thing {uuid: 'db6b11ae-114d-4f83-a044-f86505a0530a'})
				OPTIONAL MATCH (m)-[prel:HAS_MEMBER]->(p:Thing)
				OPTIONAL MATCH (m)-[orel:HAS_ORGANISATION]->(o:Thing)
				OPTIONAL MATCH (r:Thing)<-[rrel:HAS_ROLE]-(m)
				REMOVE m:Concept
				REMOVE m:Membership
				SET m={props}
				DElETE rrel, orel, prel
		`,
		Parameters: map[string]interface{}{
			"uuid": uuid,
			"props": map[string]interface{}{
				"uuid": uuid,
			},
		},
	}

	removeNodeIfUnused := &neoism.CypherQuery{
		Statement: `
				MATCH (m:Thing {uuid: {uuid}})
				OPTIONAL MATCH (m)-[a]-(x)
				WITH m, count(a) AS relCount
				WHERE relCount = 0
				DELETE m
			`,
		Parameters: map[string]interface{}{
			"uuid": uuid,
		},
	}

	err := mcd.cypherRunner.CypherBatch([]*neoism.CypherQuery{clearNode, removeNodeIfUnused})

	//s1, err := clearNode.Stats()

	if err != nil {
		return err
	}

	/*var deleted bool
	if s1.ContainsUpdates && s1.LabelsRemoved > 0 {
		deleted = true
	}*/

	return err
}

const (
	fsAuthority = "http://api.ft.com/system/FACTSET"
)
