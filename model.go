package main

import (
	"time"
)

type membership struct {
	UUID             string       `json:"uuid"`
	PrefLabel        string       `json:"prefLabel,omitempty"`
	PersonUUID       string       `json:"personUuid"`
	OrganisationUUID string       `json:"organisationUuid"`
	InceptionDate    *time.Time   `json:"inceptionDate,omitempty"`
	TerminationDate  *time.Time   `json:"terminationDate,omitempty"`
	Identifiers      []identifier `json:"identifiers,omitempty"`
	// MembershipRoleUUID string    `json:"membershipRoleUuid"`
	MembershipRoles []role `json:"membershipRoles,omitempty"`
}

type identifier struct {
	Authority       string `json:"authority"`
	IdentifierValue string `json:"identifierValue"`
}

type role struct {
	RoleUUID        string     `json:"roleuuid"`
	InceptionDate   *time.Time `json:"inceptionDate,omitempty"`
	TerminationDate *time.Time `json:"terminationDate,omitempty"`
}
