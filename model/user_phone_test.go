package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEnsurePhoneAvailableRejectsInvalid(t *testing.T) {
	err := EnsurePhoneAvailable("bad", 0)
	require.Error(t, err)
}
