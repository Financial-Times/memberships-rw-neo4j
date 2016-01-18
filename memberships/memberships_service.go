package memberships

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Financial-Times/go-fthealth/v1a"
	"github.com/Financial-Times/neo-cypher-runner-go"
	"github.com/Financial-Times/neo-utils-go"
	"github.com/jmcvetta/neoism"
	"time"
)

type CypherDriver struct {
	cypherRunner neocypherrunner.CypherRunner
	indexManager neoutils.IndexManager
}

func NewCypherDriver(cypherRunner neocypherrunner.CypherRunner, indexManager neoutils.IndexManager) CypherDriver {
	return CypherDriver{cypherRunner, indexManager}
}

func (mcd CypherDriver) Initialise() error {
	return neoutils.EnsureIndexes(mcd.indexManager, map[string]string{
		"Thing":      "uuid",
		"Concept":    "uuid",
		"Membership": "uuid"})
}

func (mcd CypherDriver) Read(uuid string) (interface{}, bool, error) {
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

	//TODO fix query to not retun a role of empty fields when there are no roles
	if len(result.MembershipRoles) == 1 && (result.MembershipRoles[0].RoleUUID == "") {
		result.MembershipRoles = make([]role, 0, 0)
	}

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

func (mcd CypherDriver) Write(thing interface{}) error {
	m := thing.(membership)

	params := map[string]interface{}{
		"uuid": m.UUID,
	}

	if m.OrganisationUUID == "" {
		errMsg := fmt.Sprintf("Organsation uuid missing  cannot create Membership with uuid=[%s]\n", m.UUID)
		log.Error(errMsg)
		return errors.New(errMsg)
	}

	if m.PersonUUID == "" {
		errMsg := fmt.Sprintf("Person uuid missing  cannot create Membership with uuid=[%s]\n", m.UUID)
		log.Error(errMsg)
		return errors.New(errMsg)
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

func (mcd CypherDriver) Delete(uuid string) (bool, error) {
	clearNode := &neoism.CypherQuery{
		Statement: `
				MATCH (m:Thing {uuid: {uuid}})
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

		IncludeStats: true,
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

	s1, err := clearNode.Stats()
	if err != nil {
		return false, err
	}

	var deleted bool
	if s1.ContainsUpdates && s1.LabelsRemoved > 0 {
		deleted = true
	}

	return deleted, err
}

func (pcd CypherDriver) DecodeJSON(dec *json.Decoder) (interface{}, string, error) {
	m := membership{}
	err := dec.Decode(&m)
	return m, m.UUID, err

}

func (pcd CypherDriver) Check() (check v1a.Check) {
	type hcUUIDResult struct {
		UUID string `json:"uuid"`
	}

	checker := func() (string, error) {
		var result []hcUUIDResult

		query := &neoism.CypherQuery{
			Statement: `MATCH (m:Membership)
					return  m.uuid as uuid
					limit 1`,
			Result: &result,
		}

		err := pcd.cypherRunner.CypherBatch([]*neoism.CypherQuery{query})

		if err != nil {
			return "", err
		}
		if len(result) == 0 {
			return "", errors.New("No Membershp found")
		}
		if result[0].UUID == "" {
			return "", errors.New("UUID not set")
		}
		return fmt.Sprintf("Found a membership with a valid uuid = %v", result[0].UUID), nil
	}

	return v1a.Check{
		BusinessImpact:   "Cannot read/write memberships via this writer",
		Name:             "Check connectivity to Neo4j - neoUrl is a parameter in hieradata for this service",
		PanicGuide:       "TODO - write panic guide",
		Severity:         1,
		TechnicalSummary: fmt.Sprintf("Cannot connect to Neo4j instance %s with at least one person loaded in it", pcd.cypherRunner),
		Checker:          checker,
	}
}

func (pcd CypherDriver) Count() (int, error) {

	results := []struct {
		Count int `json:"c"`
	}{}

	query := &neoism.CypherQuery{
		Statement: `MATCH (n:Membership) return count(n) as c`,
		Result:    &results,
	}

	err := pcd.cypherRunner.CypherBatch([]*neoism.CypherQuery{query})

	if err != nil {
		return 0, err
	}

	return results[0].Count, nil
}

const (
	fsAuthority = "http://api.ft.com/system/FACTSET"
)
