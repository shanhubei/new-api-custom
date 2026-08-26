package baidu_vod_minimax

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAccessKeys(t *testing.T) {
	ak, sk, err := ParseAccessKeys("ak123|sk456")
	require.NoError(t, err)
	assert.Equal(t, "ak123", ak)
	assert.Equal(t, "sk456", sk)

	_, _, err = ParseAccessKeys("only-one")
	require.Error(t, err)
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
