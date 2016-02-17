package memberships

import (
	"encoding/json"

	"time"

	"github.com/Financial-Times/neo-cypher-runner-go"
	"github.com/Financial-Times/neo-utils-go/neoutils"
	"github.com/jmcvetta/neoism"
)

type CypherDriver struct {
	cypherRunner neocypherrunner.CypherRunner
	indexManager neoutils.IndexManager
}

func NewCypherDriver(cypherRunner neocypherrunner.CypherRunner, indexManager neoutils.IndexManager) CypherDriver {
	return CypherDriver{cypherRunner, indexManager}
}

func (mcd CypherDriver) Initialise() error {
	return neoutils.EnsureConstraints(mcd.indexManager, map[string]string{
		"Thing":      "uuid",
		"Concept":    "uuid",
		"Membership": "uuid"})
}

func (mcd CypherDriver) Read(uuid string) (interface{}, bool, error) {
	results := []membership{}

	query := &neoism.CypherQuery{
		Statement: `
		MATCH (m:Membership {uuid:{uuid}})-[:HAS_ORGANISATION]->(o:Thing)
					OPTIONAL MATCH (p:Thing)<-[:HAS_MEMBER]-(m)
					OPTIONAL MATCH (r:Thing)<-[rr:HAS_ROLE]-(m)
					WITH p, m, o, collect({roleuuid:r.uuid,inceptionDate:rr.inceptionDate,terminationDate:rr.terminationDate }) as membershipRoles
					WITH p, m, o, membershipRoles, collect({authority:'http://api.ft.com/system/FACTSET', identifierValue: m.factsetIdentifier})as identifiers
					return m.uuid as uuid , m.prefLabel as prefLabel,m.inceptionDate as inceptionDate,
					m.terminationDate as terminationDate, o.uuid as organisationUuid, p.uuid as personUuid,membershipRoles,identifiers
					`,

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
	if len(result.Identifiers) == 1 && (result.Identifiers[0].IdentifierValue == "") {
		result.Identifiers = make([]identifier, 0, 0)
	}

	if len(result.MembershipRoles) == 1 && (result.MembershipRoles[0].RoleUUID == "") {
		result.MembershipRoles = make([]role, 0, 0)
	}

	return result, true, nil
}

func (mcd CypherDriver) Write(thing interface{}) error {
	m := thing.(membership)

	params := map[string]interface{}{
		"uuid": m.UUID,
	}

	if m.PrefLabel != "" {
		params["prefLabel"] = m.PrefLabel
	}

	if m.InceptionDate != "" {
		addDateToQueryParams(params, "inceptionDate", m.InceptionDate)
	}

	if m.TerminationDate != "" {
		addDateToQueryParams(params, "terminationDate", m.TerminationDate)
	}

	for _, identifier := range m.Identifiers {
		if identifier.Authority == fsAuthority {
			params["factsetIdentifier"] = identifier.IdentifierValue
		}
	}
	if len(m.Identifiers) == 0 {
		m.Identifiers = make([]identifier, 0, 0)
	}

	queryDelEntitiessRel := &neoism.CypherQuery{
		Statement: `MATCH (m:Thing {uuid: {uuid}})
					OPTIONAL MATCH (p:Thing)<-[rm:HAS_MEMBER]-(m)
					OPTIONAL MATCH (o:Thing)<-[ro:HAS_ORGANISATION]-(m)
					DELETE rm, ro
		`,
		Parameters: map[string]interface{}{
			"uuid": m.UUID,
		},
	}

	queries := []*neoism.CypherQuery{queryDelEntitiessRel}

	query := &neoism.CypherQuery{
		Statement: `MERGE (m:Thing {uuid: {uuid}})
					MERGE (p:Thing {uuid: {personuuid}})
					MERGE (o:Thing {uuid: {organisationuuid}})
					CREATE(m)-[:HAS_MEMBER]->(p)
		            CREATE (m)-[:HAS_ORGANISATION]->(o)
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

	queries = append(queries, query)

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

		if mr.InceptionDate != "" {
			addDateToQueryParams(rrparams, "inceptionDate", mr.InceptionDate)
		}

		if mr.TerminationDate != "" {
			addDateToQueryParams(rrparams, "terminationDate", mr.TerminationDate)
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

func (pcd CypherDriver) Check() error {
	return neoutils.Check(pcd.cypherRunner)
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

func addDateToQueryParams(params map[string]interface{}, dateName string, dateVal string) error {
	params[dateName] = dateVal
	datetimeEpoch, err := time.Parse(time.RFC3339, dateVal)
	if err != nil {
		return err
	}
	params[dateName+"Epoch"] = datetimeEpoch.Unix()
	return nil
}

const (
	fsAuthority = "http://api.ft.com/system/FACTSET"
)
