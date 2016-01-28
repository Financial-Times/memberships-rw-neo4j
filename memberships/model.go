package memberships

type membership struct {
	UUID             string       `json:"uuid"`
	PrefLabel        string       `json:"prefLabel,omitempty"`
	PersonUUID       string       `json:"personUuid"`
	OrganisationUUID string       `json:"organisationUuid"`
	InceptionDate    string       `json:"inceptionDate,omitempty"`
	TerminationDate  string       `json:"terminationDate,omitempty"`
	Identifiers      []identifier `json:"identifiers"`
	MembershipRoles  []role       `json:"membershipRoles"`
}

type identifier struct {
	Authority       string `json:"authority"`
	IdentifierValue string `json:"identifierValue"`
}

type role struct {
	RoleUUID        string `json:"roleuuid,omitempty"`
	InceptionDate   string `json:"inceptionDate,omitempty"`
	TerminationDate string `json:"terminationDate,omitempty"`
}
