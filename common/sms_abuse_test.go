package common

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAllowSmsSendMemoryCooldownAndDaily(t *testing.T) {
	oldCool, oldPhone, oldIP := SmsPhoneCooldownSeconds, SmsPhoneDailyLimit, SmsIPDailyLimit
	oldRedis := RedisEnabled
	SmsPhoneCooldownSeconds, SmsPhoneDailyLimit, SmsIPDailyLimit = 60, 2, 100
	RedisEnabled = false
	t.Cleanup(func() {
		SmsPhoneCooldownSeconds, SmsPhoneDailyLimit, SmsIPDailyLimit = oldCool, oldPhone, oldIP
		RedisEnabled = oldRedis
		smsMem.mu.Lock()
		smsMem.cooldown = make(map[string]time.Time)
		smsMem.phoneDay = make(map[string]int)
		smsMem.ipDay = make(map[string]int)
		smsMem.mu.Unlock()
	})

	phone := "13900001111"
	ip := "1.2.3.4"
	require.NoError(t, AllowSmsSend(ip, phone))
	MarkSmsSent(ip, phone)
	require.Error(t, AllowSmsSend(ip, phone))

	smsMem.mu.Lock()
	delete(smsMem.cooldown, phone)
	smsMem.mu.Unlock()

	require.NoError(t, AllowSmsSend(ip, phone))
	MarkSmsSent(ip, phone)
	smsMem.mu.Lock()
	delete(smsMem.cooldown, phone)
	smsMem.mu.Unlock()
	require.ErrorIs(t, AllowSmsSend(ip, phone), ErrSmsPhoneDaily)
}
