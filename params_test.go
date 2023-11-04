package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_fetchParams(t *testing.T) {
	params, err := fetchParams(context.Background())
	assert.NoError(t, err)
	t.Logf("%+v", params)
}
