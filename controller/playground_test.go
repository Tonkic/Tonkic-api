package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewPlaygroundTokenEnablesCrossGroupRetryForAuto(t *testing.T) {
	token := newPlaygroundToken(42, "auto")

	assert.Equal(t, 42, token.UserId)
	assert.Equal(t, "playground-auto", token.Name)
	assert.Equal(t, "auto", token.Group)
	assert.True(t, token.CrossGroupRetry)
}

func TestNewPlaygroundTokenKeepsExplicitGroupWithinGroup(t *testing.T) {
	token := newPlaygroundToken(42, "0.05羊毛")

	assert.Equal(t, "playground-0.05羊毛", token.Name)
	assert.False(t, token.CrossGroupRetry)
}
