package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetSize(t *testing.T) {
	assert := assert.New(t)
	size := getSize(3234234)

	t.Log("size", size)
	assert.Equal("3.08   M", size)
}
