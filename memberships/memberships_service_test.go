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

var membershipsDriver baseftrwapp.Service

func TestDeleteNoRoles(t *testing.T) {
	assert := assert.New(t)
	uuid := "12345"
	orgUuid := "67890"
	personUuid := "54321"
	incepDate := time.Date(2006, 1, 1, 12, 0, 0, 0, time.UTC)
	termDate := time.Date(2008, 1, 1, 12, 0, 0, 0, time.UTC)

	membershipsDriver = getMembershipsCypherDriver(t)

	membershipToDelete := membership{UUID: uuid, OrganisationUUID: orgUuid, PersonUUID: personUuid, PrefLabel: "Test label",
		InceptionDate: &incepDate, TerminationDate: &termDate,
		Identifiers:     []identifier{identifier{fsAuthority, "FACTSET_ID"}},
		MembershipRoles: make([]role, 0, 0),
	}

	assert.NoError(membershipsDriver.Write(membershipToDelete), "Failed to write membership")

	deleted, err := membershipsDriver.Delete(uuid)
	assert.True(deleted, "Didn't manage to delete membership for uuid %", uuid)
	assert.NoError(err, "Error deleting membership for uuid %s", uuid)

	_, found, err := membershipsDriver.Read(uuid)

	assert.False(found, "Found membership for uuid %s who should have been deleted", uuid)
	assert.NoError(err, "Error trying to find membership for uuid %s", uuid)
}

func TestCreateAllValuesPresent(t *testing.T) {
	assert := assert.New(t)
	uuid := "12345"
	orgUuid := "67890"
	personUuid := "54321"
	incepDate := time.Date(2006, 1, 1, 12, 0, 0, 0, time.UTC)
	termDate := time.Date(2008, 1, 1, 12, 0, 0, 0, time.UTC)
	membershipsDriver = getMembershipsCypherDriver(t)

	membershipToWrite := membership{UUID: uuid, OrganisationUUID: orgUuid, PersonUUID: personUuid, PrefLabel: "Test label",
		InceptionDate: &incepDate, TerminationDate: &termDate,
		Identifiers:     []identifier{identifier{fsAuthority, "FACTSET_ID"}},
		MembershipRoles: make([]role, 0, 0),
	}

	assert.NoError(membershipsDriver.Write(membershipToWrite), "Failed to write membership")

	readMembershipForUuidAndCheckFieldsMatch(t, uuid, membershipToWrite)

	cleanUp(t, uuid)
}

func TestCreateHandlesSpecialCharacters(t *testing.T) {
	assert := assert.New(t)
	uuid := "12345"
	orgUuid := "67890"
	personUuid := "54321"
	incepDate := time.Date(2006, 1, 1, 12, 0, 0, 0, time.UTC)
	termDate := time.Date(2008, 1, 1, 12, 0, 0, 0, time.UTC)
	membershipsDriver = getMembershipsCypherDriver(t)
	membershipToWrite := membership{UUID: uuid, OrganisationUUID: orgUuid, PersonUUID: personUuid, PrefLabel: "Test label",
		InceptionDate: &incepDate, TerminationDate: &termDate,
		Identifiers:     []identifier{identifier{fsAuthority, "FACTSET_ID"}},
		MembershipRoles: make([]role, 0, 0),
	}

	assert.NoError(membershipsDriver.Write(membershipToWrite), "Failed to write membership")

	readMembershipForUuidAndCheckFieldsMatch(t, uuid, membershipToWrite)

	cleanUp(t, uuid)
}

func TestCreateNotAllValuesPresent(t *testing.T) {
	assert := assert.New(t)
	uuid := "12345"
	orgUuid := "67890"
	personUuid := "54321"
	membershipsDriver = getMembershipsCypherDriver(t)

	membershipToWrite := membership{UUID: uuid, OrganisationUUID: orgUuid, PersonUUID: personUuid, PrefLabel: "Test label",
		Identifiers:     []identifier{identifier{fsAuthority, "FACTSET_ID"}},
		MembershipRoles: make([]role, 0, 0)}

	assert.NoError(membershipsDriver.Write(membershipToWrite), "Failed to write membership")

	readMembershipForUuidAndCheckFieldsMatch(t, uuid, membershipToWrite)

	cleanUp(t, uuid)
}

func TestUpdateWillRemovePropertiesNoLongerPresent(t *testing.T) {
	assert := assert.New(t)
	uuid := "12345"
	orgUuid := "67890"
	personUuid := "54321"
	incepDate := time.Date(2006, 1, 1, 12, 0, 0, 0, time.UTC)
	termDate := time.Date(2008, 1, 1, 12, 0, 0, 0, time.UTC)
	membershipsDriver = getMembershipsCypherDriver(t)

	membershipToWrite := membership{UUID: uuid, OrganisationUUID: orgUuid, PersonUUID: personUuid, PrefLabel: "Test label",
		InceptionDate: &incepDate, TerminationDate: &termDate,
		Identifiers:     []identifier{identifier{fsAuthority, "FACTSET_ID"}},
		MembershipRoles: make([]role, 0, 0)}

	assert.NoError(membershipsDriver.Write(membershipToWrite), "Failed to write membership")
	readMembershipForUuidAndCheckFieldsMatch(t, uuid, membershipToWrite)

	updatedMembership := membership{UUID: uuid, OrganisationUUID: orgUuid, PersonUUID: personUuid, PrefLabel: "Test2 label2",
		Identifiers: []identifier{identifier{fsAuthority, "FACTSET_ID"}}}

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
