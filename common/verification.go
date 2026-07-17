package common

import (
	"crypto/rand"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type verificationValue struct {
	code string
	time time.Time
}

const (
	EmailVerificationPurpose        = "v"
	PasswordResetPurpose            = "r"
	SmsRegisterPurpose              = "sr"
	SmsLoginPurpose                 = "sl"
	SmsBindPurpose                  = "sb"
	SmsPasswordResetPurpose         = "sp"
	NewuserPasswordResetPurpose     = "nr"  // team user email reset
	NewuserSmsPasswordResetPurpose  = "nsp" // team user SMS reset
)

var verificationMutex sync.Mutex
var verificationMap map[string]verificationValue
var verificationMapMaxSize = 10
var VerificationValidMinutes = 10

func GenerateVerificationCode(length int) string {
	code := uuid.New().String()
	code = strings.Replace(code, "-", "", -1)
	if length == 0 {
		return code
	}
	return code[:length]
}

// GenerateNumericVerificationCode returns a digit-only code for SMS templates
// that require Aliyun's "数字验证码" variable rules.
func GenerateNumericVerificationCode(length int) string {
	if length <= 0 {
		length = 6
	}
	var b strings.Builder
	b.Grow(length)
	ten := big.NewInt(10)
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, ten)
		if err != nil {
			// Extremely unlikely; fall back so SMS send can still proceed.
			n = big.NewInt(int64(i % 10))
		}
		b.WriteByte(byte('0' + n.Int64()))
	}
	return b.String()
}

func RegisterVerificationCodeWithKey(key string, code string, purpose string) {
	verificationMutex.Lock()
	defer verificationMutex.Unlock()
	verificationMap[purpose+key] = verificationValue{
		code: code,
		time: time.Now(),
	}
	if len(verificationMap) > verificationMapMaxSize {
		removeExpiredPairs()
	}
}

func VerifyCodeWithKey(key string, code string, purpose string) bool {
	verificationMutex.Lock()
	defer verificationMutex.Unlock()
	value, okay := verificationMap[purpose+key]
	now := time.Now()
	if !okay || int(now.Sub(value.time).Seconds()) >= VerificationValidMinutes*60 {
		return false
	}
	return code == value.code
}

func DeleteKey(key string, purpose string) {
	verificationMutex.Lock()
	defer verificationMutex.Unlock()
	delete(verificationMap, purpose+key)
}

// no lock inside, so the caller must lock the verificationMap before calling!
func removeExpiredPairs() {
	now := time.Now()
	for key := range verificationMap {
		if int(now.Sub(verificationMap[key].time).Seconds()) >= VerificationValidMinutes*60 {
			delete(verificationMap, key)
		}
	}
}

func init() {
	verificationMutex.Lock()
	defer verificationMutex.Unlock()
	verificationMap = make(map[string]verificationValue)
}
