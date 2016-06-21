// +build !jenkins

package memberships

import (
	"os"
	"testing"

	"github.com/Financial-Times/base-ft-rw-app-go/baseftrwapp"
	"github.com/Financial-Times/neo-utils-go/neoutils"
	"github.com/jmcvetta/neoism"
	"github.com/stretchr/testify/assert"
)

const uuid = "12345"

var minimalMembership = membership{
	UUID: uuid,
	OrganisationUUID: "",
	PersonUUID: "",
	PrefLabel: "",
	InceptionDate: "",
	TerminationDate: "",
	AlternativeIdentifiers: alternativeIdentifiers{"", make([]string, 0, 0)},
	MembershipRoles:        make([]role, 0, 0),
}
var fullMembership = membership{
	UUID: uuid,
	OrganisationUUID: "67890",
	PersonUUID: "54321",
	PrefLabel: "Test label",
	InceptionDate: "2005-01-01T00:00:00.000Z",
	TerminationDate: "2007-01-01T00:00:00.000Z",
	AlternativeIdentifiers: alternativeIdentifiers{"FACTSET_ID", []string{uuid}},
	MembershipRoles:        []role{role{"roleuuid", "2006-01-01T00:00:00.000Z", "2006-09-01T00:00:00.000Z"}},
}

var membershipsService baseftrwapp.Service

func TestDeleteMembership(t *testing.T) {
	assert := assert.New(t)
	membershipsService = getMembershipsCypherDriver(t)

	tests := []struct {
		membershipToTest membership
	}{
		{minimalMembership},
		{fullMembership},
	}
	for _, test := range tests {
		assert.NoError(membershipsService.Write(test.membershipToTest), "Failed to write membership")

		deleted, err := membershipsService.Delete(uuid)
		assert.True(deleted, "Didn't manage to delete membership for uuid %", uuid)
		assert.NoError(err, "Error deleting membership for uuid %s", uuid)

		_, found, err := membershipsService.Read(uuid)

		assert.False(found, "Found membership for uuid %s who should have been deleted", uuid)
		assert.NoError(err, "Error trying to find membership for uuid %s", uuid)
	}
}

func TestCreateMembership(t *testing.T) {
	assert := assert.New(t)
	membershipsService = getMembershipsCypherDriver(t)
	tests := []struct {
		membershipToTest membership
	}{
		{minimalMembership},
		{fullMembership},
	}
	for _, test := range tests {
		assert.NoError(membershipsService.Write(test.membershipToTest), "Failed to write membership")
		readMembershipForUuidAndCheckFieldsMatch(t, uuid, test.membershipToTest)
		cleanUpRelationshipsAndRelatedNodes(t, uuid)
		cleanUp(t, uuid)
	}
}

func TestCreateHandlesSpecialCharacters(t *testing.T) {
	assert := assert.New(t)
	membershipsService = getMembershipsCypherDriver(t)
	membershipToWrite := fullMembership
	membershipToWrite.PrefLabel = "Test's 'are' Us"
	assert.NoError(membershipsService.Write(membershipToWrite), "Failed to write membership")
	readMembershipForUuidAndCheckFieldsMatch(t, uuid, membershipToWrite)
	cleanUpRelationshipsAndRelatedNodes(t, uuid)
	cleanUp(t, uuid)
}

func TestUpdateWillRemovePropertiesNoLongerPresent(t *testing.T) {
	assert := assert.New(t)
	membershipsService = getMembershipsCypherDriver(t)
	assert.NoError(membershipsService.Write(fullMembership), "Failed to write membership")
	readMembershipForUuidAndCheckFieldsMatch(t, uuid, fullMembership)

	updatedMembership := membership{UUID: uuid, OrganisationUUID: "67890", PersonUUID: "54321", PrefLabel: "Test2 label2",
		AlternativeIdentifiers: alternativeIdentifiers{"FACTSET_ID2", make([]string, 0, 0)},
		MembershipRoles: make([]role, 0, 0)}

	assert.NoError(membershipsService.Write(updatedMembership), "Failed to write updated membership")
	readMembershipForUuidAndCheckFieldsMatch(t, uuid, updatedMembership)
	cleanUpRelationshipsAndRelatedNodes(t, uuid)
	cleanUp(t, uuid)
}

func TestUpdateWillReplaceOrgAndPerson(t *testing.T) {
	assert := assert.New(t)
	membershipsService = getMembershipsCypherDriver(t)
	assert.NoError(membershipsService.Write(fullMembership), "Failed to write membership")
	readMembershipForUuidAndCheckFieldsMatch(t, uuid, fullMembership)

	updatedMembership := membership{UUID: uuid, OrganisationUUID: "121212", PersonUUID: "323232", PrefLabel: "Test2 label2",
		AlternativeIdentifiers: alternativeIdentifiers{"FACTSET_ID2", make([]string, 0, 0)},
		MembershipRoles: make([]role, 0, 0)}

	assert.NoError(membershipsService.Write(updatedMembership), "Failed to write updated membership")
	readMembershipForUuidAndCheckFieldsMatch(t, uuid, updatedMembership)
	cleanUpRelationshipsAndRelatedNodes(t, uuid)
	cleanUp(t, uuid)
}

func TestWriteCalculateEpocCorrectly(t *testing.T) {
	assert := assert.New(t)

	membershipsService = getMembershipsCypherDriver(t)
	membershipsService.Write(fullMembership)
	membershipsCypherDriver := getMembershipsCypherDriver(t)

	result := []struct {
		MembershipInceptionDateEpoch   int `json:"m.inceptionDateEpoch"`
		MembershipTerminationDateEpoch int `json:"m.terminationDateEpoch"`
		RoleInceptionDateEpoch         int `json:"rr.inceptionDateEpoch"`
		RoleTerminationDateEpoch       int `json:"rr.terminationDateEpoch"`
	}{}

	getEpocQuery := &neoism.CypherQuery{
		Statement: `
		MATCH (m:Membership {uuid:'12345'})
			   OPTIONAL MATCH (r:Thing)<-[rr:HAS_ROLE]-(m)
               return  m.inceptionDateEpoch, m.terminationDateEpoch , rr.inceptionDateEpoch, rr.terminationDateEpoch  
		`,
		Result: &result,
	}

	err := membershipsCypherDriver.cypherRunner.CypherBatch([]*neoism.CypherQuery{getEpocQuery})
	assert.NoError(err)
	assert.Equal(1104537600, result[0].MembershipInceptionDateEpoch, "Epoc of 2005-01-01T01:00:00.000Z should be 1104537600")
	assert.Equal(1167609600, result[0].MembershipTerminationDateEpoch, "Epoc of 2007-01-01T01:00:00.000Z should be 1167609600")
	assert.Equal(1136073600, result[0].RoleInceptionDateEpoch, "Epoc of  2006-01-01T01:00:00.000Z should be 1136073600")
	assert.Equal(1157068800, result[0].RoleTerminationDateEpoch, "Epoc of 2006-09-01T01:00:00.000Z should be 1157068800")
	cleanUp(t, uuid)
}

func getMembershipsCypherDriver(t *testing.T) CypherDriver {
	assert := assert.New(t)
	url := os.Getenv("NEO4J_TEST_URL")
	if url == "" {
		url = "http://localhost:7474/db/data"
	}
	db, err := neoism.Connect(url)
	assert.NoError(err, "Failed to connect to Neo4j")
	return NewCypherDriver(neoutils.StringerDb{db}, db)
}

func readMembershipForUuidAndCheckFieldsMatch(t *testing.T, uuid string, expectedMembership membership) {
	assert := assert.New(t)
	storedMembership, found, err := membershipsService.Read(uuid)

	assert.NoError(err, "Error finding membership for uuid %s", uuid)
	assert.True(found, "Didn't find membership for uuid %s", uuid)
	assert.Equal(expectedMembership, storedMembership, "memberships should be the same")
}

func cleanUp(t *testing.T, uuid string) {
	assert := assert.New(t)
	deleted, err := membershipsService.Delete(uuid)
	assert.True(deleted, "Didn't manage to delete person for uuid %", uuid)
	assert.NoError(err, "Error deleting membership for uuid %s", uuid)
}

func cleanUpRelationshipsAndRelatedNodes(t *testing.T, uuid string) {
	assert := assert.New(t)
	membershipsCypherDriver := getMembershipsCypherDriver(t)
	query := &neoism.CypherQuery{
		Statement: `
				MATCH (m:Thing {uuid: {muuid}})
				OPTIONAL MATCH (n)<-[rel]-(m)
			    delete rel, n
			`,
		Parameters: map[string]interface{}{
			"muuid": uuid,
		},
	}
	assert.NoError(membershipsCypherDriver.cypherRunner.CypherBatch([]*neoism.CypherQuery{query}), "Error deleting membership for uuid %s", uuid)
}
