package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizePhone(t *testing.T) {
	tests := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"13800138000", "13800138000", false},
		{" 13800138000 ", "13800138000", false},
		{"+8613800138000", "13800138000", false},
		{"8613800138000", "13800138000", false},
		{"1380013800", "", true},
		{"23800138000", "", true},
		{"", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := NormalizePhone(tt.in)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
