package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEstimateTokenPreservesRepresentativeResults(t *testing.T) {
	tests := []struct {
		name     string
		provider Provider
		text     string
		want     int
	}{
		{name: "empty", provider: OpenAI, text: "", want: 0},
		{name: "English words", provider: OpenAI, text: "hello world", want: 3},
		{name: "math symbols", provider: OpenAI, text: "∑√", want: 6},
		{name: "URL delimiters", provider: OpenAI, text: "/:?", want: 3},
		{name: "unknown provider fallback", provider: Unknown, text: "hello world", want: 3},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, EstimateToken(test.provider, test.text))
		})
	}
}
