package common

import (
	"errors"
	"regexp"
	"strings"
)

var cnMobileRegexp = regexp.MustCompile(`^1[3-9]\d{9}$`)

func NormalizePhone(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "-", "")
	if strings.HasPrefix(s, "+86") {
		s = s[3:]
	} else if strings.HasPrefix(s, "86") && len(s) == 13 {
		s = s[2:]
	}
	if !cnMobileRegexp.MatchString(s) {
		return "", errors.New("invalid phone number")
	}
	return s, nil
}

func MaskPhone(phone string) string {
	if len(phone) != 11 {
		return "***"
	}
	return phone[:3] + "****" + phone[7:]
}
