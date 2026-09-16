# Review Package Task 1
BASE: 6548aa478715a3c7952b7fdf7f1351d9341a772a
HEAD: ff76b2d4cb503fb07e60158484799b73abc4bd2e

## Commits
ff76b2d4 feat(baidu-vod-minimax): add BCE bce-auth-v1 signer

## Stat
 relay/channel/baidu_vod_minimax/sign.go      | 134 +++++++++++++++++++++++++++
 relay/channel/baidu_vod_minimax/sign_test.go |  56 +++++++++++
 2 files changed, 190 insertions(+)

## Diff
```
diff --git a/relay/channel/baidu_vod_minimax/sign.go b/relay/channel/baidu_vod_minimax/sign.go
new file mode 100644
index 00000000..88316b3a
--- /dev/null
+++ b/relay/channel/baidu_vod_minimax/sign.go
@@ -0,0 +1,134 @@
+package baidu_vod_minimax
+
+import (
+	"crypto/hmac"
+	"crypto/sha256"
+	"encoding/hex"
+	"fmt"
+	"net/http"
+	"net/url"
+	"sort"
+	"strings"
+	"time"
+)
+
+const defaultExpirationSeconds = 1800
+
+func ParseAccessKeys(apiKey string) (string, string, error) {
+	parts := strings.Split(apiKey, "|")
+	if len(parts) != 2 {
+		return "", "", fmt.Errorf("invalid api_key, required format is accessKey|secretKey")
+	}
+	ak := strings.TrimSpace(parts[0])
+	sk := strings.TrimSpace(parts[1])
+	if ak == "" || sk == "" {
+		return "", "", fmt.Errorf("invalid api_key, required format is accessKey|secretKey")
+	}
+	return ak, sk, nil
+}
+
+func uriEncode(s string, encodeSlash bool) string {
+	var b strings.Builder
+	for i := 0; i < len(s); i++ {
+		c := s[i]
+		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') ||
+			c == '_' || c == '-' || c == '~' || c == '.' {
+			b.WriteByte(c)
+		} else if c == '/' {
+			if encodeSlash {
+				b.WriteString("%2F")
+			} else {
+				b.WriteByte(c)
+			}
+		} else {
+			b.WriteString(fmt.Sprintf("%%%02X", c))
+		}
+	}
+	return b.String()
+}
+
+func hmacSHA256Hex(key []byte, msg string) string {
+	m := hmac.New(sha256.New, key)
+	m.Write([]byte(msg))
+	return hex.EncodeToString(m.Sum(nil))
+}
+
+func SignAuthorization(ak, sk, method, host, canonicalURI, canonicalQuery string, now time.Time, expirationSec int) (string, error) {
+	if expirationSec <= 0 {
+		expirationSec = defaultExpirationSeconds
+	}
+	if canonicalURI == "" {
+		canonicalURI = "/"
+	}
+	if !strings.HasPrefix(canonicalURI, "/") {
+		canonicalURI = "/" + canonicalURI
+	}
+	timestamp := now.UTC().Format("2006-01-02T15:04:05Z")
+	authPrefix := fmt.Sprintf("bce-auth-v1/%s/%s/%d", ak, timestamp, expirationSec)
+	canonicalURIEnc := uriEncode(canonicalURI, false)
+	canonicalHeaders := "host:" + uriEncode(host, true)
+	canonicalRequest := strings.ToUpper(method) + "\n" +
+		canonicalURIEnc + "\n" +
+		canonicalQuery + "\n" +
+		canonicalHeaders
+	signingKey := hmacSHA256Hex([]byte(sk), authPrefix)
+	signature := hmacSHA256Hex([]byte(signingKey), canonicalRequest)
+	return fmt.Sprintf("%s/host/%s", authPrefix, signature), nil
+}
+
+func ApplyBCEAuth(header *http.Header, apiKey, method, requestURL string, now time.Time) error {
+	if header == nil {
+		return fmt.Errorf("header is nil")
+	}
+	ak, sk, err := ParseAccessKeys(apiKey)
+	if err != nil {
+		return err
+	}
+	u, err := url.Parse(requestURL)
+	if err != nil {
+		return fmt.Errorf("parse request url: %w", err)
+	}
+	host := u.Host
+	if host == "" {
+		return fmt.Errorf("request url missing host")
+	}
+	path := u.Path
+	if path == "" {
+		path = "/"
+	}
+	canonicalQuery := canonicalizeQuery(u.Query())
+	auth, err := SignAuthorization(ak, sk, method, host, path, canonicalQuery, now, defaultExpirationSeconds)
+	if err != nil {
+		return err
+	}
+	header.Set("Host", host)
+	header.Set("Authorization", auth)
+	return nil
+}
+
+func canonicalizeQuery(values url.Values) string {
+	if len(values) == 0 {
+		return ""
+	}
+	keys := make([]string, 0, len(values))
+	for k := range values {
+		if strings.EqualFold(k, "authorization") {
+			continue
+		}
+		keys = append(keys, k)
+	}
+	sort.Strings(keys)
+	parts := make([]string, 0, len(keys))
+	for _, k := range keys {
+		vs := values[k]
+		if len(vs) == 0 {
+			parts = append(parts, uriEncode(k, true)+"=")
+			continue
+		}
+		for _, v := range vs {
+			parts = append(parts, uriEncode(k, true)+"="+uriEncode(v, true))
+		}
+	}
+	sort.Strings(parts)
+	return strings.Join(parts, "&")
+}
diff --git a/relay/channel/baidu_vod_minimax/sign_test.go b/relay/channel/baidu_vod_minimax/sign_test.go
new file mode 100644
index 00000000..aa572cb0
--- /dev/null
+++ b/relay/channel/baidu_vod_minimax/sign_test.go
@@ -0,0 +1,56 @@
+package baidu_vod_minimax
+
+import (
+	"net/http"
+	"strings"
+	"testing"
+	"time"
+
+	"github.com/stretchr/testify/assert"
+	"github.com/stretchr/testify/require"
+)
+
+func TestParseAccessKeys(t *testing.T) {
+	ak, sk, err := ParseAccessKeys("ak123|sk456")
+	require.NoError(t, err)
+	assert.Equal(t, "ak123", ak)
+	assert.Equal(t, "sk456", sk)
+
+	_, _, err = ParseAccessKeys("only-one")
+	require.Error(t, err)
+}
+
+func TestSignAuthorizationContainsHostSignedHeader(t *testing.T) {
+	now := time.Date(2026, 6, 12, 2, 45, 13, 0, time.UTC)
+	auth, err := SignAuthorization(
+		"ak-test", "sk-test",
+		http.MethodPost, "vod.bj.baidubce.com", "/v2/tts", "",
+		now, 1800,
+	)
+	require.NoError(t, err)
+	assert.True(t, strings.HasPrefix(auth, "bce-auth-v1/ak-test/2026-06-12T02:45:13Z/1800/host/"))
+	assert.NotContains(t, auth, "Bearer")
+	parts := strings.Split(auth, "/")
+	require.Len(t, parts, 6)
+	assert.Equal(t, "host", parts[4])
+	assert.Regexp(t, "^[0-9a-f]{64}$", parts[5])
+}
+
+func TestSignAuthorizationDeterministic(t *testing.T) {
+	now := time.Date(2026, 6, 12, 2, 45, 13, 0, time.UTC)
+	a, err := SignAuthorization("ak", "sk", http.MethodPost, "vod.bj.baidubce.com", "/v2/tts", "", now, 1800)
+	require.NoError(t, err)
+	b, err := SignAuthorization("ak", "sk", http.MethodPost, "vod.bj.baidubce.com", "/v2/tts", "", now, 1800)
+	require.NoError(t, err)
+	assert.Equal(t, a, b)
+}
+
+func TestApplyBCEAuthSetsHostAndAuthorization(t *testing.T) {
+	h := make(http.Header)
+	now := time.Date(2026, 6, 12, 2, 45, 13, 0, time.UTC)
+	err := ApplyBCEAuth(&h, "ak|sk", http.MethodPost, "https://vod.bj.baidubce.com/v2/tts", now)
+	require.NoError(t, err)
+	assert.Equal(t, "vod.bj.baidubce.com", h.Get("Host"))
+	assert.True(t, strings.HasPrefix(h.Get("Authorization"), "bce-auth-v1/"))
+	assert.Contains(t, h.Get("Authorization"), "/host/")
+}

```
