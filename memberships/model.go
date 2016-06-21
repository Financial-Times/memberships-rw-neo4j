package memberships

type membership struct {
	UUID                   string                 `json:"uuid"`
	PrefLabel              string                 `json:"prefLabel,omitempty"`
	PersonUUID             string                 `json:"personUuid"`
	OrganisationUUID       string                 `json:"organisationUuid"`
	InceptionDate          string                 `json:"inceptionDate,omitempty"`
	TerminationDate        string                 `json:"terminationDate,omitempty"`
	AlternativeIdentifiers alternativeIdentifiers `json:"alternativeIdentifiers"`
	MembershipRoles        []role                 `json:"membershipRoles"`
}

type alternativeIdentifiers struct {
	FactsetIdentifier string   `json:"factsetIdentifier,omitempty"`
	UUIDS             []string `json:"uuids"`
}

const (
	uppIdentifierLabel     = "UPPIdentifier"
	factsetIdentifierLabel = "FactsetIdentifier"
)

type role struct {
	RoleUUID        string `json:"roleuuid,omitempty"`
	InceptionDate   string `json:"inceptionDate,omitempty"`
	TerminationDate string `json:"terminationDate,omitempty"`
}
