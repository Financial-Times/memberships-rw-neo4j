#!/usr/bin/env bash

curl -s -i -XPUT -H "X-Request-Id: 123" -H "Content-Type: application/json" localhost:8080/memberships/g10e101c-dbcf-356f-929e-669573defa56 --data '{
    "uuid": "g10e101c-dbcf-356f-929e-669573defa56",
    "prefLabel": "Partner",
    "personUuid": "4cd0e22e-6db4-3768-aa18-8c91e6090a9d",
    "organisationUuid": "c75c10df-0bbc-3c61-8e8e-30a2ff0042aa",
    "inceptionDate": "2004-01-01T00:00:00.000Z",
    "terminationDate": "2005-01-01T00:00:00.000Z",
    "alternativeIdentifiers": {
            "factsetIdentifier": "12345",
            "uuids": ["28198957-954f-4810-b2cd-843f1f6150b7"]
    },
    "membershipRoles": [
        {
            "roleUuid": "c9c9cae0-afe6-30ca-b7e6-fe5cd9dccb60",
            "inceptionDate": "2004-01-01T00:00:00.000Z"
        },
        {
            "roleUuid": "eaa6f59e-b24c-3d36-8b79-062381f828e0",
            "inceptionDate": "2004-01-01T00:00:00.000Z",
            "terminationDate": "2005-01-01T00:00:00.000Z"
        },
        {
            "roleUuid": "73b9c823-aa25-33b7-abe8-1e6d830ded55",
            "inceptionDate": "2004-01-01T00:00:00.000Z"
        }
    ]
}'
