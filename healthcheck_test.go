package main

import (
	"errors"
	"github.com/jmcvetta/neoism"
	"github.com/stretchr/testify/assert"
	"testing"
)

const neoUrl = "http://localhost:7474/db/data"

func TestHealthCheckSuccess(t *testing.T) {
	assert := assert.New(t)
	runner := hcMockRunner{result: hcUUIDResult{UUID: "e80c286f-aa90-465c-a41b-281ff9b8bad3"}, returnResult: true}
	hc := setUpHealthCheck(runner, neoUrl)
	_, err := hc.Checker()
	assert.NoError(err)
}

func TestHealthNoResult(t *testing.T) {
	assert := assert.New(t)
	runner := hcMockRunner{returnResult: false}
	hc := setUpHealthCheck(runner, neoUrl)
	_, err := hc.Checker()
	assert.Error(err)
}

func TestHealthCheckNoUUID(t *testing.T) {
	assert := assert.New(t)
	runner := hcMockRunner{result: hcUUIDResult{UUID: ""}, returnResult: true}
	hc := setUpHealthCheck(runner, neoUrl)
	_, err := hc.Checker()
	assert.Error(err)
}

func TestHealthCheckPropagateError(t *testing.T) {
	assert := assert.New(t)
	theError := errors.New("expected error")
	runner := hcMockRunner{err: theError}
	hc := setUpHealthCheck(runner, neoUrl)
	_, err := hc.Checker()
	assert.Equal(theError, err)
}

type hcMockRunner struct {
	result       hcUUIDResult
	returnResult bool
	err          error
}

func (sdb hcMockRunner) CypherBatch(queries []*neoism.CypherQuery) error {
	if sdb.err != nil {
		return sdb.err
	}

	if len(queries) != 1 {
		return errors.New("expected 1 query")
	}

	res := (queries[0].Result).(*[]hcUUIDResult)

	if sdb.returnResult {
		*res = append(*res, sdb.result)
	}

	return nil
}
