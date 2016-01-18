// +build !jenkins

package memberships

import (
	"os"
	"testing"

	"github.com/Financial-Times/base-ft-rw-app-go"
	"github.com/Financial-Times/neo-utils-go"
	"github.com/jmcvetta/neoism"
	"github.com/stretchr/testify/assert"
	"time"
)

const uuid = "12345"

var incepDate = time.Date(2006, 1, 1, 12, 0, 0, 0, time.UTC)
var termDate = time.Date(2008, 1, 1, 12, 0, 0, 0, time.UTC)

var minimalMembership = membership{UUID: uuid, OrganisationUUID: "", PersonUUID: "", PrefLabel: "",
	InceptionDate: nil, TerminationDate: nil,
	Identifiers:     make([]identifier, 0, 0),
	MembershipRoles: make([]role, 0, 0),
}
var fullMembership = membership{UUID: uuid, OrganisationUUID: "67890", PersonUUID: "54321", PrefLabel: "Test label",
	InceptionDate: &incepDate, TerminationDate: &termDate,
	Identifiers:     []identifier{identifier{fsAuthority, "FACTSET_ID"}},
	MembershipRoles: []role{role{"roleuuid", &incepDate, &termDate}},
}

var membershipsDriver baseftrwapp.Service

func TestDeleteMembership(t *testing.T) {
	assert := assert.New(t)
	membershipsDriver = getMembershipsCypherDriver(t)

	tests := []struct {
		membershipToTest membership
	}{
		{minimalMembership},
		{fullMembership},
	}
	for _, test := range tests {
		assert.NoError(membershipsDriver.Write(test.membershipToTest), "Failed to write membership")

		deleted, err := membershipsDriver.Delete(uuid)
		assert.True(deleted, "Didn't manage to delete membership for uuid %", uuid)
		assert.NoError(err, "Error deleting membership for uuid %s", uuid)

		_, found, err := membershipsDriver.Read(uuid)

		assert.False(found, "Found membership for uuid %s who should have been deleted", uuid)
		assert.NoError(err, "Error trying to find membership for uuid %s", uuid)
	}
}

func TestCreateMembership(t *testing.T) {
	assert := assert.New(t)
	membershipsDriver = getMembershipsCypherDriver(t)
	tests := []struct {
		membershipToTest membership
	}{
		{minimalMembership},
		{fullMembership},
	}
	for _, test := range tests {
		assert.NoError(membershipsDriver.Write(test.membershipToTest), "Failed to write membership")
		readMembershipForUuidAndCheckFieldsMatch(t, uuid, test.membershipToTest)
		cleanUp(t, uuid)
	}
}

func TestCreateHandlesSpecialCharacters(t *testing.T) {
	assert := assert.New(t)
	membershipsDriver = getMembershipsCypherDriver(t)
	membershipToWrite := fullMembership
	membershipToWrite.PrefLabel = "Test's 'are' Us"
	assert.NoError(membershipsDriver.Write(membershipToWrite), "Failed to write membership")
	readMembershipForUuidAndCheckFieldsMatch(t, uuid, membershipToWrite)
	cleanUp(t, uuid)
}

func TestUpdateWillRemovePropertiesNoLongerPresent(t *testing.T) {
	assert := assert.New(t)
	membershipsDriver = getMembershipsCypherDriver(t)
	assert.NoError(membershipsDriver.Write(fullMembership), "Failed to write membership")
	readMembershipForUuidAndCheckFieldsMatch(t, uuid, fullMembership)

	updatedMembership := membership{UUID: uuid, OrganisationUUID: "678901", PersonUUID: "54321", PrefLabel: "Test2 label2",
		Identifiers:     []identifier{identifier{fsAuthority, "FACTSET_ID2"}},
		MembershipRoles: make([]role, 0, 0)}

	assert.NoError(membershipsDriver.Write(updatedMembership), "Failed to write updated membership")
	readMembershipForUuidAndCheckFieldsMatch(t, uuid, updatedMembership)
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
	storedMembership, found, err := membershipsDriver.Read(uuid)

	assert.NoError(err, "Error finding membership for uuid %s", uuid)
	assert.True(found, "Didn't find membership for uuid %s", uuid)
	assert.Equal(expectedMembership, storedMembership, "memberships should be the same")
}

func cleanUp(t *testing.T, uuid string) {
	assert := assert.New(t)
	deleted, err := membershipsDriver.Delete(uuid)
	assert.True(deleted, "Didn't manage to delete person for uuid %", uuid)
	assert.NoError(err, "Error deleting membership for uuid %s", uuid)
}
