package baidu_vod_minimax

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const goldenAuthorization = "bce-auth-v1/ak-test/2026-06-12T02:45:13Z/1800/host/b6952868b8ae3da6a73cd732e90d620f23f6ae3ecce40832c97fdcd729f8902f"

func TestParseAccessKeys(t *testing.T) {
	ak, sk, err := ParseAccessKeys("ak123|sk456")
	require.NoError(t, err)
	assert.Equal(t, "ak123", ak)
	assert.Equal(t, "sk456", sk)

	_, _, err = ParseAccessKeys("only-one")
	require.Error(t, err)
}

func TestSignAuthorizationGoldenVector(t *testing.T) {
	now := time.Date(2026, 6, 12, 2, 45, 13, 0, time.UTC)
	auth, err := SignAuthorization(
		"ak-test", "sk-test",
		http.MethodPost, "vod.bj.baidubce.com", "/v2/tts", "",
		now, 1800,
	)
	require.NoError(t, err)

	authPrefix := "bce-auth-v1/ak-test/2026-06-12T02:45:13Z/1800"
	canonicalRequest := "POST\n/v2/tts\n\nhost:vod.bj.baidubce.com"
	signingKeyMAC := hmac.New(sha256.New, []byte("sk-test"))
	signingKeyMAC.Write([]byte(authPrefix))
	signingKey := hex.EncodeToString(signingKeyMAC.Sum(nil))
	sigMAC := hmac.New(sha256.New, []byte(signingKey))
	sigMAC.Write([]byte(canonicalRequest))
	expectedSig := hex.EncodeToString(sigMAC.Sum(nil))
	expectedAuth := fmt.Sprintf("%s/host/%s", authPrefix, expectedSig)

	assert.Equal(t, expectedAuth, auth)
	assert.Equal(t, goldenAuthorization, auth)
}

func TestSignAuthorizationTrimsHost(t *testing.T) {
	now := time.Date(2026, 6, 12, 2, 45, 13, 0, time.UTC)
	withSpace, err := SignAuthorization(
		"ak-test", "sk-test",
		http.MethodPost, "  vod.bj.baidubce.com  ", "/v2/tts", "",
		now, 1800,
	)
	require.NoError(t, err)
	withoutSpace, err := SignAuthorization(
		"ak-test", "sk-test",
		http.MethodPost, "vod.bj.baidubce.com", "/v2/tts", "",
		now, 1800,
	)
	require.NoError(t, err)
	assert.Equal(t, withoutSpace, withSpace)
	assert.Equal(t, goldenAuthorization, withSpace)
}

func TestSignAuthorizationContainsHostSignedHeader(t *testing.T) {
	now := time.Date(2026, 6, 12, 2, 45, 13, 0, time.UTC)
	auth, err := SignAuthorization(
		"ak-test", "sk-test",
		http.MethodPost, "vod.bj.baidubce.com", "/v2/tts", "",
		now, 1800,
	)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(auth, "bce-auth-v1/ak-test/2026-06-12T02:45:13Z/1800/host/"))
	assert.NotContains(t, auth, "Bearer")
	parts := strings.Split(auth, "/")
	require.Len(t, parts, 6)
	assert.Equal(t, "host", parts[4])
	assert.Regexp(t, "^[0-9a-f]{64}$", parts[5])
}

func TestSignAuthorizationDeterministic(t *testing.T) {
	now := time.Date(2026, 6, 12, 2, 45, 13, 0, time.UTC)
	a, err := SignAuthorization("ak", "sk", http.MethodPost, "vod.bj.baidubce.com", "/v2/tts", "", now, 1800)
	require.NoError(t, err)
	b, err := SignAuthorization("ak", "sk", http.MethodPost, "vod.bj.baidubce.com", "/v2/tts", "", now, 1800)
	require.NoError(t, err)
	assert.Equal(t, a, b)
}

func TestApplyBCEAuthSetsHostAndAuthorization(t *testing.T) {
	h := make(http.Header)
	now := time.Date(2026, 6, 12, 2, 45, 13, 0, time.UTC)
	err := ApplyBCEAuth(&h, "ak|sk", http.MethodPost, "https://vod.bj.baidubce.com/v2/tts", now)
	require.NoError(t, err)
	assert.Equal(t, "vod.bj.baidubce.com", h.Get("Host"))
	assert.True(t, strings.HasPrefix(h.Get("Authorization"), "bce-auth-v1/"))
	assert.Contains(t, h.Get("Authorization"), "/host/")
}
