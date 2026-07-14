package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecideRegisterContact(t *testing.T) {
	tests := []struct {
		name         string
		emailEnabled bool
		smsEnabled   bool
		email        string
		phone        string
		wantMode     registerContactMode
		wantErr      bool
	}{
		{
			name:         "neither enabled",
			emailEnabled: false,
			smsEnabled:   false,
			wantMode:     registerContactNone,
		},
		{
			name:         "email only missing email",
			emailEnabled: true,
			smsEnabled:   false,
			wantErr:      true,
		},
		{
			name:         "email only with email",
			emailEnabled: true,
			smsEnabled:   false,
			email:        "user@example.com",
			wantMode:     registerContactEmail,
		},
		{
			name:         "sms only missing phone",
			emailEnabled: false,
			smsEnabled:   true,
			wantErr:      true,
		},
		{
			name:         "sms only with phone",
			emailEnabled: false,
			smsEnabled:   true,
			phone:        "13800138000",
			wantMode:     registerContactPhone,
		},
		{
			name:         "both enabled neither provided",
			emailEnabled: true,
			smsEnabled:   true,
			wantErr:      true,
		},
		{
			name:         "both enabled email only",
			emailEnabled: true,
			smsEnabled:   true,
			email:        "user@example.com",
			wantMode:     registerContactEmail,
		},
		{
			name:         "both enabled phone only",
			emailEnabled: true,
			smsEnabled:   true,
			phone:        "13800138000",
			wantMode:     registerContactPhone,
		},
		{
			name:         "both enabled prefer phone",
			emailEnabled: true,
			smsEnabled:   true,
			email:        "user@example.com",
			phone:        "13800138000",
			wantMode:     registerContactPhone,
		},
		{
			name:         "both enabled trims whitespace phone",
			emailEnabled: true,
			smsEnabled:   true,
			email:        "user@example.com",
			phone:        " 13800138000 ",
			wantMode:     registerContactPhone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMode, err := decideRegisterContact(tt.emailEnabled, tt.smsEnabled, tt.email, tt.phone)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantMode, gotMode)
		})
	}
}
