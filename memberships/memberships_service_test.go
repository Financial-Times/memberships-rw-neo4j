// +build !jenkins

package memberships

import (
	"fmt"
	"os"
	"sort"
	"testing"

	"github.com/Financial-Times/base-ft-rw-app-go/baseftrwapp"
	"github.com/Financial-Times/neo-utils-go/neoutils"
	"github.com/jmcvetta/neoism"
	"github.com/stretchr/testify/assert"
)

const (
	membershipUUID string = "79e4af29-9911-4cd0-860c-884dc2c33af6"
	personUUID     string = "2bf87e91-a4de-4759-b646-291d21d9d485"
	orgUUID        string = "4e6e4584-9a60-4320-a84b-d6fd234737cf"
	roleUUID       string = "22416992-aa7e-47dc-9dd2-bdf877e4b877"
	newPersonUUID  string = "11111111-1111-1111-1111-111111111111"
	newOrgUUID     string = "22222222-2222-2222-2222-222222222222"
)

var fullMembership = membership{
	UUID:                   membershipUUID,
	OrganisationUUID:       orgUUID,
	PersonUUID:             personUUID,
	PrefLabel:              "Test label",
	InceptionDate:          "2005-01-01T00:00:00.000Z",
	TerminationDate:        "2007-01-01T00:00:00.000Z",
	AlternativeIdentifiers: alternativeIdentifiers{"FACTSET_ID", []string{membershipUUID}},
	MembershipRoles:        []role{role{roleUUID, "2006-01-01T00:00:00.000Z", "2006-09-01T00:00:00.000Z"}},
}

var membershipsService baseftrwapp.Service

func TestCreateFullMembership(t *testing.T) {
	assert := assert.New(t)

	db := getDatabaseConnectionAndCheckClean(t, assert)
	membershipDriver := getCypherDriver(db)
	defer cleanDB(db, t, assert)

	assert.NoError(membershipDriver.Write(fullMembership), "Failed to write membership")
	readMembershipAndCompare(fullMembership, t, db)
}

func TestDeleteMembership(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	membershipDriver := getCypherDriver(db)
	defer cleanDB(db, t, assert)

	assert.NoError(membershipDriver.Write(fullMembership), "Failed to write membership")

	found, err := membershipDriver.Delete(membershipUUID)
	assert.True(found, "Didn't manage to delete membership for uuid %", membershipUUID)
	assert.NoError(err, "Error deleting membership for uuid %s", membershipUUID)

	m, found, err := membershipDriver.Read(membershipUUID)

	assert.Equal(membership{}, m, "Found membership %s who should have been deleted", m)
	assert.False(found, "Found membership for uuid %s who should have been deleted", membershipUUID)
	assert.NoError(err, "Error trying to find membership for uuid %s", membershipUUID)
}

func TestCreateHandlesSpecialCharacters(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	membershipDriver := getCypherDriver(db)
	defer cleanDB(db, t, assert)

	membershipToWrite := membership{UUID: membershipUUID, PrefLabel: "Engineér", PersonUUID: personUUID, OrganisationUUID: orgUUID, AlternativeIdentifiers: alternativeIdentifiers{FactsetIdentifier: "FACTSET_ID", UUIDS: []string{membershipUUID}}, MembershipRoles: []role{role{roleUUID, "2006-01-01T00:00:00.000Z", "2006-09-01T00:00:00.000Z"}}}

	assert.NoError(membershipDriver.Write(membershipToWrite), "Failed to write membership")

	readMembershipAndCompare(membershipToWrite, t, db)
}

func TestUpdateWillRemovePropertiesNoLongerPresent(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	membershipDriver := getCypherDriver(db)
	defer cleanDB(db, t, assert)

	assert.NoError(membershipDriver.Write(fullMembership), "Failed to write membership")
	storedFullMembership, _, err := membershipDriver.Read(membershipUUID)

	assert.NoError(err)
	assert.NotEmpty(storedFullMembership)

	var minimalMembership = membership{
		UUID:                   membershipUUID,
		OrganisationUUID:       orgUUID,
		PersonUUID:             personUUID,
		AlternativeIdentifiers: alternativeIdentifiers{"FACTSET_ID", []string{membershipUUID}},
		MembershipRoles:        []role{role{roleUUID, "value1", "value2"}},
	}

	assert.NoError(membershipDriver.Write(minimalMembership), "Failed to write updated membership")

	readMembershipAndCompare(minimalMembership, t, db)
}

func TestUpdateWillReplaceOrgAndPerson(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	membershipDriver := getCypherDriver(db)
	defer cleanDB(db, t, assert)

	assert.NoError(membershipDriver.Write(fullMembership), "Failed to write membership")
	storedFullMembership, _, err := membershipDriver.Read(membershipUUID)

	assert.NoError(err)
	assert.NotEmpty(storedFullMembership)

	var updatedMembership = membership{
		UUID:                   membershipUUID,
		OrganisationUUID:       newOrgUUID,
		PersonUUID:             newPersonUUID,
		AlternativeIdentifiers: alternativeIdentifiers{"FACTSET_ID", []string{membershipUUID}},
		MembershipRoles:        []role{role{roleUUID, "value1", "value2"}},
	}

	assert.NoError(membershipDriver.Write(updatedMembership), "Failed to write updated membership")

	readMembershipAndCompare(updatedMembership, t, db)
}

func TestWriteCalculateEpocCorrectly(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert)
	membershipDriver := getCypherDriver(db)
	defer cleanDB(db, t, assert)

	assert.NoError(membershipDriver.Write(fullMembership), "Failed to write membership")

	result := []struct {
		MembershipInceptionDateEpoch   int `json:"m.inceptionDateEpoch"`
		MembershipTerminationDateEpoch int `json:"m.terminationDateEpoch"`
		RoleInceptionDateEpoch         int `json:"rr.inceptionDateEpoch"`
		RoleTerminationDateEpoch       int `json:"rr.terminationDateEpoch"`
	}{}

	getEpocQuery := &neoism.CypherQuery{
		Statement: `
		MATCH (m:Membership {uuid:'79e4af29-9911-4cd0-860c-884dc2c33af6'})
			   OPTIONAL MATCH (r:Thing)<-[rr:HAS_ROLE]-(m)
               return  m.inceptionDateEpoch, m.terminationDateEpoch , rr.inceptionDateEpoch, rr.terminationDateEpoch
		`,
		Result: &result,
	}

	err := membershipDriver.conn.CypherBatch([]*neoism.CypherQuery{getEpocQuery})
	assert.NoError(err)
	assert.Equal(1104537600, result[0].MembershipInceptionDateEpoch, "Epoc of 2005-01-01T01:00:00.000Z should be 1104537600")
	assert.Equal(1167609600, result[0].MembershipTerminationDateEpoch, "Epoc of 2007-01-01T01:00:00.000Z should be 1167609600")
	assert.Equal(1136073600, result[0].RoleInceptionDateEpoch, "Epoc of  2006-01-01T01:00:00.000Z should be 1136073600")
	assert.Equal(1157068800, result[0].RoleTerminationDateEpoch, "Epoc of 2006-09-01T01:00:00.000Z should be 1157068800")
}

func getDatabaseConnection(assert *assert.Assertions) neoutils.NeoConnection {
	url := os.Getenv("NEO4J_TEST_URL")
	if url == "" {
		url = "http://localhost:7474/db/data"
	}

	conf := neoutils.DefaultConnectionConfig()
	conf.Transactional = false
	db, err := neoutils.Connect(url, conf)
	assert.NoError(err, "Failed to connect to Neo4j")
	return db
}

func getCypherDriver(db neoutils.NeoConnection) service {
	cr := NewCypherMembershipService(db)
	cr.Initialise()
	return cr
}

func getDatabaseConnectionAndCheckClean(t *testing.T, assert *assert.Assertions) neoutils.NeoConnection {
	db := getDatabaseConnection(assert)
	cleanDB(db, t, assert)
	checkDbClean(db, t)
	return db
}

func cleanDB(db neoutils.NeoConnection, t *testing.T, assert *assert.Assertions) {
	qs := []*neoism.CypherQuery{
		{
			Statement: fmt.Sprintf("MATCH (fp:Thing {uuid: '%v'})<-[:IDENTIFIES*0..]-(i:Identifier) DETACH DELETE fp, i", membershipUUID),
		},
		{
			Statement: fmt.Sprintf("MATCH (mp:Thing {uuid: '%v'})<-[:IDENTIFIES*0..]-(i:Identifier) DETACH DELETE mp, i", roleUUID),
		},
		{
			Statement: fmt.Sprintf("MATCH (fp:Thing {uuid: '%v'})<-[:IDENTIFIES*0..]-(i:Identifier) DETACH DELETE fp, i", personUUID),
		},
		{
			Statement: fmt.Sprintf("MATCH (fp:Thing {uuid: '%v'})<-[:IDENTIFIES*0..]-(i:Identifier) DETACH DELETE fp, i", orgUUID),
		},
		{
			Statement: fmt.Sprintf("MATCH (fp:Thing {uuid: '%v'})<-[:IDENTIFIES*0..]-(i:Identifier) DETACH DELETE fp, i", newPersonUUID),
		},
		{
			Statement: fmt.Sprintf("MATCH (fp:Thing {uuid: '%v'})<-[:IDENTIFIES*0..]-(i:Identifier) DETACH DELETE fp, i", newOrgUUID),
		},
	}

	err := db.CypherBatch(qs)
	assert.NoError(err)
}

func checkDbClean(db neoutils.NeoConnection, t *testing.T) {
	assert := assert.New(t)

	result := []struct {
		Uuid string `json:"m.uuid"`
	}{}

	checkGraph := neoism.CypherQuery{
		Statement: `
			MATCH (m:Thing) WHERE m.uuid in {uuids} RETURN m.uuid
		`,
		Parameters: neoism.Props{
			"uuids": []string{roleUUID, personUUID, orgUUID, membershipUUID},
		},
		Result: &result,
	}
	err := db.CypherBatch([]*neoism.CypherQuery{&checkGraph})
	assert.NoError(err)
	assert.Empty(result)
}

func readMembershipAndCompare(expected membership, t *testing.T, db neoutils.NeoConnection) {
	sort.Strings(expected.AlternativeIdentifiers.UUIDS)

	actual, found, err := getCypherDriver(db).Read(expected.UUID)
	assert.NoError(t, err)
	assert.True(t, found)

	actualMembership := actual.(membership)
	sort.Strings(actualMembership.AlternativeIdentifiers.UUIDS)

	assert.EqualValues(t, expected, actualMembership)
}
