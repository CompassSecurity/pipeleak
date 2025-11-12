package github

import (
	"testing"

	pkggithub "github.com/CompassSecurity/pipeleak/pkg/github"
	"github.com/stretchr/testify/assert"
)

func TestDeleteHighestXKeys(t *testing.T) {
	tests := []struct {
		name     string
		input    map[int64]struct{}
		nrKeys   int
		expected map[int64]struct{}
	}{
		{
			name: "delete highest 2 keys from 5",
			input: map[int64]struct{}{
				1: {}, 2: {}, 3: {}, 4: {}, 5: {},
			},
			nrKeys: 2,
			expected: map[int64]struct{}{
				1: {}, 2: {}, 3: {},
			},
		},
		{
			name: "delete all keys when nrKeys equals map size",
			input: map[int64]struct{}{
				10: {}, 20: {}, 30: {},
			},
			nrKeys:   3,
			expected: map[int64]struct{}{},
		},
		{
			name: "return empty map when nrKeys exceeds map size",
			input: map[int64]struct{}{
				1: {}, 2: {},
			},
			nrKeys:   5,
			expected: map[int64]struct{}{},
		},
		{
			name: "delete nothing when nrKeys is 0",
			input: map[int64]struct{}{
				100: {}, 200: {}, 300: {},
			},
			nrKeys: 0,
			expected: map[int64]struct{}{
				100: {}, 200: {}, 300: {},
			},
		},
		{
			name:     "handle empty map",
			input:    map[int64]struct{}{},
			nrKeys:   1,
			expected: map[int64]struct{}{},
		},
		{
			name: "delete single highest key",
			input: map[int64]struct{}{
				5: {}, 10: {}, 15: {}, 20: {},
			},
			nrKeys: 1,
			expected: map[int64]struct{}{
				5: {}, 10: {}, 15: {},
			},
		},
		{
			name: "handle negative keys correctly",
			input: map[int64]struct{}{
				-10: {}, -5: {}, 0: {}, 5: {}, 10: {},
			},
			nrKeys: 2,
			expected: map[int64]struct{}{
				-10: {}, -5: {}, 0: {},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pkggithub.DeleteHighestXKeys(tt.input, tt.nrKeys)
			assert.Equal(t, tt.expected, result, "Result map should match expected")
		})
	}
}
