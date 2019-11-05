package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetSize(t *testing.T) {
	assert := assert.New(t)

	actual := getSize(323)
	expected := "323    b"
	assert.Equal(expected, actual)

	actual = getSize(3234234)
	expected = "3.08   M"
	assert.Equal(expected, actual)

	actual = getSize(3234)
	expected = "3.16   K"
	assert.Equal(expected, actual)

	actual = getSize(322345234634)
	expected = "300.21 G"
	assert.Equal(expected, actual)
}
