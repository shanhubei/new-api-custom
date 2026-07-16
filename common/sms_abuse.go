package common

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrSmsPhoneCooldown = errors.New("sms send too frequent for this phone")
	ErrSmsPhoneDaily    = errors.New("sms daily limit exceeded for this phone")
	ErrSmsIPDaily       = errors.New("sms daily limit exceeded for this ip")
)

type smsMemoryCounter struct {
	mu       sync.Mutex
	cooldown map[string]time.Time // phone -> next allowed
	phoneDay map[string]int       // phone:yyyy-mm-dd -> count
	ipDay    map[string]int       // ip:yyyy-mm-dd -> count
}

var smsMem = &smsMemoryCounter{
	cooldown: make(map[string]time.Time),
	phoneDay: make(map[string]int),
	ipDay:    make(map[string]int),
}

func smsDayKey(prefix, id string) string {
	return fmt.Sprintf("%s:%s:%s", prefix, id, time.Now().Format("20060102"))
}

func smsCooldownSeconds() int {
	if SmsPhoneCooldownSeconds <= 0 {
		return 60
	}
	return SmsPhoneCooldownSeconds
}

func smsPhoneDailyLimit() int {
	if SmsPhoneDailyLimit <= 0 {
		return 10
	}
	return SmsPhoneDailyLimit
}

func smsIPDailyLimit() int {
	if SmsIPDailyLimit <= 0 {
		return 40
	}
	return SmsIPDailyLimit
}

// AllowSmsSend checks per-phone cooldown and daily caps (IP + phone).
// Call before SendAliyunSms. Does not reserve quota until MarkSmsSent.
func AllowSmsSend(ip, phone string) error {
	return allowSmsSend(ip, phone)
}

func allowSmsSend(ip, phone string) error {
	if phone == "" {
		return errors.New("invalid phone number")
	}
	if RedisEnabled && RDB != nil {
		return allowSmsSendRedis(ip, phone)
	}
	return allowSmsSendMemory(ip, phone)
}

func allowSmsSendRedis(ip, phone string) error {
	ctx := context.Background()
	rdb := RDB
	coolKey := "sms:cooldown:" + phone
	ttl, err := rdb.TTL(ctx, coolKey).Result()
	if err == nil && ttl > 0 {
		return fmt.Errorf("%w, retry after %d seconds", ErrSmsPhoneCooldown, int(ttl.Seconds())+1)
	}
	phoneKey := smsDayKey("sms:daily:phone", phone)
	phoneCount, _ := rdb.Get(ctx, phoneKey).Int()
	if phoneCount >= smsPhoneDailyLimit() {
		return ErrSmsPhoneDaily
	}
	if ip != "" {
		ipKey := smsDayKey("sms:daily:ip", ip)
		ipCount, _ := rdb.Get(ctx, ipKey).Int()
		if ipCount >= smsIPDailyLimit() {
			return ErrSmsIPDaily
		}
	}
	return nil
}

func allowSmsSendMemory(ip, phone string) error {
	smsMem.mu.Lock()
	defer smsMem.mu.Unlock()
	now := time.Now()
	if next, ok := smsMem.cooldown[phone]; ok && now.Before(next) {
		sec := int(next.Sub(now).Seconds()) + 1
		return fmt.Errorf("%w, retry after %d seconds", ErrSmsPhoneCooldown, sec)
	}
	phoneKey := smsDayKey("phone", phone)
	if smsMem.phoneDay[phoneKey] >= smsPhoneDailyLimit() {
		return ErrSmsPhoneDaily
	}
	if ip != "" {
		ipKey := smsDayKey("ip", ip)
		if smsMem.ipDay[ipKey] >= smsIPDailyLimit() {
			return ErrSmsIPDaily
		}
	}
	return nil
}

// MarkSmsSent records a successful Aliyun send for cooldown and daily counters.
func MarkSmsSent(ip, phone string) {
	if phone == "" {
		return
	}
	if RedisEnabled && RDB != nil {
		markSmsSentRedis(ip, phone)
		return
	}
	markSmsSentMemory(ip, phone)
}

func markSmsSentRedis(ip, phone string) {
	ctx := context.Background()
	rdb := RDB
	cool := time.Duration(smsCooldownSeconds()) * time.Second
	_ = rdb.Set(ctx, "sms:cooldown:"+phone, "1", cool).Err()

	phoneKey := smsDayKey("sms:daily:phone", phone)
	if n, err := rdb.Incr(ctx, phoneKey).Result(); err == nil && n == 1 {
		_ = rdb.Expire(ctx, phoneKey, 26*time.Hour).Err()
	}
	if ip != "" {
		ipKey := smsDayKey("sms:daily:ip", ip)
		if n, err := rdb.Incr(ctx, ipKey).Result(); err == nil && n == 1 {
			_ = rdb.Expire(ctx, ipKey, 26*time.Hour).Err()
		}
	}
}

func markSmsSentMemory(ip, phone string) {
	smsMem.mu.Lock()
	defer smsMem.mu.Unlock()
	smsMem.cooldown[phone] = time.Now().Add(time.Duration(smsCooldownSeconds()) * time.Second)
	phoneKey := smsDayKey("phone", phone)
	smsMem.phoneDay[phoneKey]++
	if ip != "" {
		ipKey := smsDayKey("ip", ip)
		smsMem.ipDay[ipKey]++
	}
}

// SmsAbuseMessage maps guard errors to user-facing API messages.
func SmsAbuseMessage(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, ErrSmsPhoneCooldown) {
		return err.Error()
	}
	if errors.Is(err, ErrSmsPhoneDaily) {
		return "今日该手机号短信发送次数已达上限"
	}
	if errors.Is(err, ErrSmsIPDaily) {
		return "今日当前网络短信发送次数已达上限"
	}
	return err.Error()
}
