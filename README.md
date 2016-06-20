Memberships Reader/Writer for Neo4j (memberships-rw-neo4j)
==========================================================

__An API for reading/writing memberships into Neo4j. Expects the memberships json supplied to be in the format
that comes out of the memberships transformer.__

[![CircleCI](https://circleci.com/gh/Financial-Times/memberships-rw-neo4j.svg?style=svg)](https://circleci.com/gh/Financial-Times/memberships-rw-neo4j)


Installation
------------

For the first time:

`go get github.com/Financial-Times/memberships-rw-neo4j`

or update:

`go get -u github.com/Financial-Times/memberships-rw-neo4j`


Running
-------

`$GOPATH/bin/memberships-rw-neo4j --neo-url={neo4jUrl} --port={port} --batchSize=50 --timeoutMs=20`

All arguments are optional, they default to a local Neo4j install on the default port (7474), application running on port 8080,
batchSize of 1024 and timeoutMs of 50. NB: the default batchSize is much higher than the throughput the instance data
ingester currently can cope with.


Updating the model
------------------

Use gojson against a transformer endpoint to create a person struct and update the model.go file. NB: we DO need a separate identifier struct

`curl http://ftaps35629-law1a-eu-t:8080/transformers/memberships/g10e101c-dbcf-356f-929e-669573defa56`


Building
--------

This service is built and deployed via Jenkins.

<a href="http://ftjen10085-lvpr-uk-p:8181/view/JOBS-memberships-rw-neo4j/job/mrwn-memberships-rw-neo4j-build/">Build job</a><br>
<a href="http://ftjen10085-lvpr-uk-p:8181/view/JOBS-memberships-rw-neo4j/job/mrwn-memberships-rw-neo4j-deploy-test/">Deploy Test</a><br>
<a href="http://ftjen10085-lvpr-uk-p:8181/view/JOBS-memberships-rw-neo4j/job/mrwn-memberships-rw-neo4j-deploy-prod/">Deploy Prod</a>

The build works via git tags. To prepare a new release
- update the version in /puppet/ft-memberships_rw_neo4j/Modulefile, e.g. to 0.0.12
- git tag that commit using `git tag 0.0.12`
- `git push --tags`

The deploy also works via git tag and you can also select the environment to deploy to.


Try it!
-------

Note the data in the example (e.g. UUIDs) is not real.

PUT example:

`curl -XPUT -H "X-Request-Id: 123" -H "Content-Type: application/json" localhost:8080/memberships/g10e101c-dbcf-356f-929e-669573defa56 --data '{
    "uuid": "g10e101c-dbcf-356f-929e-669573defa56",
    "prefLabel": "Partner",
    "personUuid": "4cd0e22e-6db4-3768-aa18-8c91e6090a9d",
    "organisationUuid": "c75c10df-0bbc-3c61-8e8e-30a2ff0042aa",
    "inceptionDate": "2004-01-01T00:00:00.000Z",
    "terminationDate": "2005-01-01T00:00:00.000Z",
    "identifiers": [
        {
            "authority": "http://api.ft.com/system/FACTSET",
            "identifierValue": "100007"
        }
    ],
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
}`


GET example:

`curl -H "X-Request-Id: 123" localhost:8080/memberships/3fa70485-3a57-3b9b-9449-774b001cd965`

DELETE example:

`curl -H "X-Request-Id: 123" localhost:8080/memberships/3fa70485-3a57-3b9b-9449-774b001cd965`


Healthchecks: [http://localhost:8080/__health](http://localhost:8080/__health)

Good-to-go: [http://localhost:8080/__gtg](http://localhost:8080/__gtg)

Ping: [http://localhost:8080/ping](http://localhost:8080/ping)

### Logging

The application uses [logrus](https://github.com/Sirupsen/logrus), the log file is initialised in [main.go](main.go).
Logging requires an env app parameter, for all environments other than local logs are written to file.
When running locally, logging is written to console (if you want to log locally to file you need to pass in an env parameter
that is != `local`.)
 
