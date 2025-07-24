package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCleanup(t *testing.T) {
	expected := []int{4, 3, 2, 1, 0}
	out := []int{}
	cfs := CleanupFuncs{}
	for i := range 5 {
		cfs.Defer(func() error {
			out = append(out, i)
			return nil
		})
	}

	err := cfs.Cleanup()
	assert.NoError(t, err)

	require.Len(t, out, 5)
	for i := range expected {
		assert.Equal(t, expected[i], out[i])
	}
}
