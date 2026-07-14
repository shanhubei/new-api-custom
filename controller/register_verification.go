package controller

import (
	"errors"
	"strings"
)

type registerContactMode int

const (
	registerContactNone registerContactMode = iota
	registerContactEmail
	registerContactPhone
)

// decideRegisterContact implements E/S table from the spec.
// preferPhone when both email and phone non-empty under E&&S.
func decideRegisterContact(emailEnabled, smsEnabled bool, email, phone string) (registerContactMode, error) {
	email = strings.TrimSpace(email)
	phone = strings.TrimSpace(phone)
	switch {
	case !emailEnabled && !smsEnabled:
		return registerContactNone, nil
	case emailEnabled && !smsEnabled:
		if email == "" {
			return 0, errors.New("email required")
		}
		return registerContactEmail, nil
	case !emailEnabled && smsEnabled:
		if phone == "" {
			return 0, errors.New("phone required")
		}
		return registerContactPhone, nil
	default: // both
		if phone != "" {
			return registerContactPhone, nil
		}
		if email != "" {
			return registerContactEmail, nil
		}
		return 0, errors.New("email or phone required")
	}
}
