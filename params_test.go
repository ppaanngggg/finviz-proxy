package main

import (
	"context"
	"testing"
)

func Test_fetchParams(t *testing.T) {
	params, err := fetchParams(context.Background())
	if err != nil {
		t.Error(err)
	}
	t.Logf("%+v", params)
}
