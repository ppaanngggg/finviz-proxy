package pkg

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_parseTable(t *testing.T) {
	// read file screener.ashx.html
	page, err := os.ReadFile("screener.ashx")
	assert.NoError(t, err)
	table, err := parseTable(page)
	assert.NoError(t, err)
	j, err := json.MarshalIndent(table, "", "  ")
	assert.NoError(t, err)
	println(string(j))
}
