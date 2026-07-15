package common

import (
	"testing"
	"unicode"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateNumericVerificationCode(t *testing.T) {
	code := GenerateNumericVerificationCode(6)
	require.Len(t, code, 6)
	for _, r := range code {
		assert.True(t, unicode.IsDigit(r), "expected digit, got %q in %q", r, code)
	}

	emptyLen := GenerateNumericVerificationCode(0)
	require.Len(t, emptyLen, 6)
}
