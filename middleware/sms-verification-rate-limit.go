package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/QuantumNous/new-api/common"

	"github.com/gin-gonic/gin"
)

const SmsVerificationRateLimitMark = "SV"

func smsIPMaxRequests() int {
	if common.SmsIPMaxRequests <= 0 {
		return 2
	}
	return common.SmsIPMaxRequests
}

func smsIPWindowSeconds() int {
	if common.SmsIPWindowSeconds <= 0 {
		return 30
	}
	return common.SmsIPWindowSeconds
}

func redisSmsVerificationRateLimiter(c *gin.Context) {
	ctx := context.Background()
	rdb := common.RDB
	maxReq := smsIPMaxRequests()
	window := smsIPWindowSeconds()
	key := "smsVerification:" + SmsVerificationRateLimitMark + ":" + c.ClientIP()

	count, err := rdb.Incr(ctx, key).Result()
	if err != nil {
		memorySmsVerificationRateLimiter(c)
		return
	}

	if count == 1 {
		_ = rdb.Expire(ctx, key, time.Duration(window)*time.Second).Err()
	}

	if count <= int64(maxReq) {
		c.Next()
		return
	}

	ttl, err := rdb.TTL(ctx, key).Result()
	waitSeconds := int64(window)
	if err == nil && ttl > 0 {
		waitSeconds = int64(ttl.Seconds())
	}

	c.JSON(http.StatusTooManyRequests, gin.H{
		"success": false,
		"message": fmt.Sprintf("发送过于频繁，请等待 %d 秒后再试", waitSeconds),
	})
	c.Abort()
}

func memorySmsVerificationRateLimiter(c *gin.Context) {
	key := SmsVerificationRateLimitMark + ":" + c.ClientIP()
	maxReq := smsIPMaxRequests()
	window := smsIPWindowSeconds()

	if !inMemoryRateLimiter.Request(key, maxReq, int64(window)) {
		c.JSON(http.StatusTooManyRequests, gin.H{
			"success": false,
			"message": "发送过于频繁，请稍后再试",
		})
		c.Abort()
		return
	}

	c.Next()
}

func SmsVerificationRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		if common.RedisEnabled {
			redisSmsVerificationRateLimiter(c)
		} else {
			inMemoryRateLimiter.Init(common.RateLimitKeyExpirationDuration)
			memorySmsVerificationRateLimiter(c)
		}
	}
}
