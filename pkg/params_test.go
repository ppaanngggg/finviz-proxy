package pkg

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_fetchParams(t *testing.T) {
	params, err := FetchParams(context.Background())
	assert.NoError(t, err)
	j, err := json.MarshalIndent(params, "", "  ")
	assert.NoError(t, err)
	println(string(j))
}
