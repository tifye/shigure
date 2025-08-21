package activity

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseISO8601Duration(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
	}{
		{
			input:    "PT60M20S",
			expected: time.Duration(time.Minute*60 + time.Second*20),
		},
		{
			input:    "PT24H20S",
			expected: time.Duration(time.Hour*24 + time.Second*20),
		},
		{
			input:    "PT20S",
			expected: time.Duration(time.Second * 20),
		},
		{
			input:    "PT60M",
			expected: time.Duration(time.Minute * 60),
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			dur := parseYoutubeVideoDuration(tt.input)
			assert.Equal(t, tt.expected, dur)
		})
	}
}
