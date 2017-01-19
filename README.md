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

1. Download the code.
2. Build it:

        go install

3. Download, install and start neo4j (otherwise the app will panic and exit).
3. Run it:

        $GOPATH/bin/memberships-rw-neo4j --neo-url={neo4jUrl} --port={port} --batchSize=50 --timeoutMs=20

All arguments are optional, they default to a local Neo4j install on the default port (7474), application running on port 8080,
batchSize of 1024 and timeoutMs of 50. NB: the default `batchSize` is much higher than the throughput the instance data
ingester currently can cope with.


Updating the model
------------------

Use [gojson](https://github.com/ChimeraCoder/gojson) against a transformer endpoint to create a person struct and update the
[model.go](memberships/model.go) file. NB: we DO need a separate identifier struct.

`curl http://ftaps35629-law1a-eu-t:8080/transformers/memberships/g10e101c-dbcf-356f-929e-669573defa56`


Building
--------

This service is built and deployed via Jenkins. The build chain can be found at
[JOBS-memberships-rw-neo4j](http://ftjen10085-lvpr-uk-p.osb.ft.com:8181/view/JOBS-memberships-rw-neo4j).

The build works via git tags. To prepare a new release:

1. update the version in the [Modulefile](/puppet/ft-memberships_rw_neo4j/Modulefile), e.g. to 0.0.12.
2. git tag that commit using `git tag 0.0.12`
3. `git push --tags`

The deploy also works via git tag and you can also select the environment to deploy to.


Try it!
-------

Note the data in the example (e.g. UUIDs) is not real.

* PUT example: see [test_put.bash](test_put.bash):

        ./test_put.bash

* GET example piping to `jq` (using the same UUID as the `PUT` request above): see [test_get.bash](test_get.bash):

        ./test_get.bash | jq '.'

* DELETE example:

        curl -s -H "X-Request-Id: 123" localhost:8080/memberships/g10e101c-dbcf-356f-929e-669573defa56 | jq '.'

* Health checks: [http://localhost:8080/__health](http://localhost:8080/__health)

* Good-to-go: [http://localhost:8080/__gtg](http://localhost:8080/__gtg)

* Ping: [http://localhost:8080/__ping](http://localhost:8080/__ping) or [http://localhost:8080/ping](http://localhost:8080/ping)  

* Build Info: [http://localhost:8080/__build-info](http://localhost:8080/__build-info) or [http://localhost:8080/build-info](http://localhost:8080/build-info) 


### View your data in the database

    MATCH (m:Membership {uuid:"g10e101c-dbcf-356f-929e-669573defa56"}) RETURN m


### Logging

The application uses [logrus](https://github.com/Sirupsen/logrus), the log file is initialised in [main.go](main.go).
Logging requires an `env` app parameter, for all environments other than local logs are written to file.
When running locally, logging is written to console (if you want to log locally to file you need to pass in an env parameter
that is != `local`.)
 
