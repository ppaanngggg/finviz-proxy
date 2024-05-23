package pkg

import (
	"context"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func Test_fetchFinvizPage(t *testing.T) {
	html, err := fetchFinvizPage(context.Background(), "")
	assert.NoError(t, err)
	os.WriteFile("screener.ashx", html, 0644)
}
