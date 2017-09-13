package memberships

import (
	"encoding/json"

	"time"

	"fmt"
	"github.com/Financial-Times/neo-utils-go/neoutils"
	log "github.com/sirupsen/logrus"
	"github.com/jmcvetta/neoism"
)

type service struct {
	conn neoutils.NeoConnection
}

func NewCypherMembershipService(cypherRunner neoutils.NeoConnection) service {
	return service{cypherRunner}
}

func (s service) Initialise() error {

	err := s.conn.EnsureIndexes(map[string]string{
		"Identifier": "value",
	})

	if err != nil {
		return err
	}

	return s.conn.EnsureConstraints(map[string]string{
		"Thing":             "uuid",
		"Concept":           "uuid",
		"Membership":        "uuid",
		"FactsetIdentifier": "value",
		"UPPIdentifier":     "value"})
}

func (s service) Read(uuid string) (interface{}, bool, error) {
	results := []membership{}

	query := &neoism.CypherQuery{
		Statement: `
		MATCH (m:Membership {uuid:{uuid}})-[:HAS_ORGANISATION]->(o:Thing)
					OPTIONAL MATCH (p:Thing)<-[:HAS_MEMBER]-(m)
					OPTIONAL MATCH (r:Thing)<-[rr:HAS_ROLE]-(m)
					OPTIONAL MATCH (upp:UPPIdentifier)-[:IDENTIFIES]->(m)
					OPTIONAL MATCH (fs:FactsetIdentifier)-[:IDENTIFIES]->(m)
					WITH p, m, o, upp, fs, collect({roleuuid:r.uuid,inceptionDate:rr.inceptionDate,terminationDate:rr.terminationDate}) as membershipRoles
					return
						m.uuid as uuid,
						m.prefLabel as prefLabel,
						m.inceptionDate as inceptionDate,
						m.terminationDate as terminationDate,
						o.uuid as organisationUuid,
						p.uuid as personUuid,
						membershipRoles,
						{uuids:collect(distinct upp.value), factsetIdentifier:fs.value} as alternativeIdentifiers`,

		Parameters: map[string]interface{}{
			"uuid": uuid,
		},
		Result: &results,
	}
	err := s.conn.CypherBatch([]*neoism.CypherQuery{query})

	if err != nil {
		return membership{}, false, err
	}

	if len(results) == 0 {
		return membership{}, false, nil
	}

	result := results[0]

	log.WithFields(log.Fields{"result_count": result}).Debug("Returning results")

	if len(result.MembershipRoles) == 1 && (result.MembershipRoles[0].RoleUUID == "") {
		result.MembershipRoles = make([]role, 0, 0)
	}

	return result, true, nil
}

func (s service) Write(thing interface{}) error {
	m := thing.(membership)

	queries := []*neoism.CypherQuery{}

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

	//cleanUP all the previous IDENTIFIERS referring to that uuid
	deletePreviousIdentifiersQuery := &neoism.CypherQuery{
		Statement: `MATCH (t:Thing {uuid:{uuid}})
		OPTIONAL MATCH (t)<-[iden:IDENTIFIES]-(i)
		DELETE iden, i`,
		Parameters: map[string]interface{}{
			"uuid": m.UUID,
		},
	}
	queries = append(queries, deletePreviousIdentifiersQuery)

	queryDelEntitiesRel := &neoism.CypherQuery{
		Statement: `MATCH (m:Thing {uuid: {uuid}})
					OPTIONAL MATCH (p:Thing)<-[rm:HAS_MEMBER]-(m)
					OPTIONAL MATCH (o:Thing)<-[ro:HAS_ORGANISATION]-(m)
					DELETE rm, ro
		`,
		Parameters: map[string]interface{}{
			"uuid": m.UUID,
		},
	}
	queries = append(queries, queryDelEntitiesRel)

	if m.AlternativeIdentifiers.FactsetIdentifier != "" {
		log.Debug("Creating FactsetIdentifier query")
		q := createNewIdentifierQuery(
			m.UUID,
			factsetIdentifierLabel,
			m.AlternativeIdentifiers.FactsetIdentifier,
		)
		queries = append(queries, q)
	}

	for _, alternativeUUID := range m.AlternativeIdentifiers.UUIDS {
		log.Debug("Processing alternative UUID")
		q := createNewIdentifierQuery(m.UUID, uppIdentifierLabel, alternativeUUID)
		queries = append(queries, q)
	}

	createMembershipQuery := &neoism.CypherQuery{
		Statement: `MERGE (m:Thing	 {uuid: {uuid}})
			    MERGE (personUPP:Identifier:UPPIdentifier{value:{personuuid}})
                            MERGE (personUPP)-[:IDENTIFIES]->(p:Thing) ON CREATE SET p.uuid = {personuuid}
			    MERGE (orgUPP:Identifier:UPPIdentifier{value:{organisationuuid}})
                            MERGE (orgUPP)-[:IDENTIFIES]->(o:Thing) ON CREATE SET o.uuid = {organisationuuid}
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

	queries = append(queries, createMembershipQuery)

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

		q := &neoism.CypherQuery{
			Statement: `
				MERGE (m:Thing {uuid:{muuid}})
				MERGE (roleUPP:Identifier:UPPIdentifier{value:{ruuid}})
                           	MERGE (roleUPP)-[:IDENTIFIES]->(r:Thing) ON CREATE SET r.uuid = {ruuid}
				CREATE (m)-[rel:HAS_ROLE]->(r)
				SET rel={rrparams}
			`,
			Parameters: map[string]interface{}{
				"muuid":    m.UUID,
				"ruuid":    mr.RoleUUID,
				"rrparams": rrparams,
			},
		}

		queries = append(queries, q)
	}
	log.WithFields(log.Fields{"query_count": len(queries)}).Debug("Executing queries...")
	return s.conn.CypherBatch(queries)
}

func createNewIdentifierQuery(uuid string, identifierLabel string, identifierValue string) *neoism.CypherQuery {
	statementTemplate := fmt.Sprintf(`MERGE (t:Thing {uuid:{uuid}})
					CREATE (i:Identifier {value:{value}})
					MERGE (t)<-[:IDENTIFIES]-(i)
					set i : %s `, identifierLabel)
	query := &neoism.CypherQuery{
		Statement: statementTemplate,
		Parameters: map[string]interface{}{
			"uuid":  uuid,
			"value": identifierValue,
		},
	}
	return query
}

func (s service) Delete(uuid string) (bool, error) {
	clearNode := &neoism.CypherQuery{
		Statement: `
				MATCH (m:Thing {uuid: {uuid}})
				OPTIONAL MATCH (m)-[prel:HAS_MEMBER]->(p:Thing)
				OPTIONAL MATCH (m)-[orel:HAS_ORGANISATION]->(o:Thing)
				OPTIONAL MATCH (r:Thing)<-[rrel:HAS_ROLE]-(m)
				OPTIONAL MATCH (m)<-[iden:IDENTIFIES]-(i:Identifier)
				REMOVE m:Concept
				REMOVE m:Membership
				DELETE iden, i
				SET m={props}
				DELETE rrel, orel, prel
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

	err := s.conn.CypherBatch([]*neoism.CypherQuery{clearNode, removeNodeIfUnused})

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

func (s service) DecodeJSON(dec *json.Decoder) (interface{}, string, error) {
	m := membership{}
	err := dec.Decode(&m)
	return m, m.UUID, err

}

func (s service) Check() error {
	return neoutils.Check(s.conn)
}

func (s service) Count() (int, error) {

	results := []struct {
		Count int `json:"c"`
	}{}

	query := &neoism.CypherQuery{
		Statement: `MATCH (n:Membership) return count(n) as c`,
		Result:    &results,
	}

	err := s.conn.CypherBatch([]*neoism.CypherQuery{query})

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
