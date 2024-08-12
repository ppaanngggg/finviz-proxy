package pkg

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_FetchAllFutures(t *testing.T) {
	futures, err := FetchAllFutures(context.Background(), false)
	assert.NoError(t, err)
	assert.NotNil(t, futures)
	assert.NotEmpty(t, futures)
}
