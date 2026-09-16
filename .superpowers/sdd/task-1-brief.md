### Task 1: BCE `bce-auth-v1` signer

**Files:**
- Create: `relay/channel/baidu_vod_minimax/sign.go`
- Create: `relay/channel/baidu_vod_minimax/sign_test.go`

**Interfaces:**
- Produces:
  - `ParseAccessKeys(apiKey string) (ak, sk string, err error)`
  - `SignAuthorization(ak, sk, method, host, canonicalURI, canonicalQuery string, now time.Time, expirationSec int) (authorization string, err error)`
  - `ApplyBCEAuth(header *http.Header, apiKey, method, requestURL string, now time.Time) error` 鈥?sets `Host` + `Authorization`

- [ ] **Step 1: Write failing tests**

```go
package baidu_vod_minimax

import (
	"net/http"
	"net/url"
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
```

- [ ] **Step 2: Run tests 鈥?expect FAIL**

```bash
go test ./relay/channel/baidu_vod_minimax/ -run "TestParseAccessKeys|TestSignAuthorization|TestApplyBCEAuth" -count=1
```

Expected: FAIL (package/functions undefined)

- [ ] **Step 3: Implement `sign.go`**

Implement BCE v1 per https://cloud.baidu.com/doc/Reference/s/njwvz1yfu :

```go
package baidu_vod_minimax

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

const defaultExpirationSeconds = 1800

func ParseAccessKeys(apiKey string) (string, string, error) {
	parts := strings.Split(apiKey, "|")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid api_key, required format is accessKey|secretKey")
	}
	ak := strings.TrimSpace(parts[0])
	sk := strings.TrimSpace(parts[1])
	if ak == "" || sk == "" {
		return "", "", fmt.Errorf("invalid api_key, required format is accessKey|secretKey")
	}
	return ak, sk, nil
}

func uriEncode(s string, encodeSlash bool) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') ||
			c == '_' || c == '-' || c == '~' || c == '.' {
			b.WriteByte(c)
		} else if c == '/' {
			if encodeSlash {
				b.WriteString("%2F")
			} else {
				b.WriteByte(c)
			}
		} else {
			b.WriteString(fmt.Sprintf("%%%02X", c))
		}
	}
	return b.String()
}

func hmacSHA256Hex(key []byte, msg string) string {
	m := hmac.New(sha256.New, key)
	m.Write([]byte(msg))
	return hex.EncodeToString(m.Sum(nil))
}

func SignAuthorization(ak, sk, method, host, canonicalURI, canonicalQuery string, now time.Time, expirationSec int) (string, error) {
	if expirationSec <= 0 {
		expirationSec = defaultExpirationSeconds
	}
	if canonicalURI == "" {
		canonicalURI = "/"
	}
	if !strings.HasPrefix(canonicalURI, "/") {
		canonicalURI = "/" + canonicalURI
	}
	timestamp := now.UTC().Format("2006-01-02T15:04:05Z")
	authPrefix := fmt.Sprintf("bce-auth-v1/%s/%s/%d", ak, timestamp, expirationSec)
	canonicalURIEnc := uriEncode(canonicalURI, false)
	canonicalHeaders := "host:" + uriEncode(strings.TrimSpace(strings.ToLower(host)), true)
	// If host value encoding: doc says UriEncode(name)+":"+UriEncode(value).
	// Host name typically has no reserved chars; use uriEncode(host, true) on value only:
	canonicalHeaders = "host:" + uriEncode(host, true)
	canonicalRequest := strings.ToUpper(method) + "\n" +
		canonicalURIEnc + "\n" +
		canonicalQuery + "\n" +
		canonicalHeaders
	signingKey := hmacSHA256Hex([]byte(sk), authPrefix)
	signature := hmacSHA256Hex([]byte(signingKey), canonicalRequest)
	return fmt.Sprintf("%s/host/%s", authPrefix, signature), nil
}

func ApplyBCEAuth(header *http.Header, apiKey, method, requestURL string, now time.Time) error {
	if header == nil {
		return fmt.Errorf("header is nil")
	}
	ak, sk, err := ParseAccessKeys(apiKey)
	if err != nil {
		return err
	}
	u, err := url.Parse(requestURL)
	if err != nil {
		return fmt.Errorf("parse request url: %w", err)
	}
	host := u.Host
	if host == "" {
		return fmt.Errorf("request url missing host")
	}
	path := u.EscapedPath()
	if path == "" {
		path = "/"
	}
	// Build canonical query: sorted UriEncode(k)=UriEncode(v), skip authorization
	canonicalQuery := canonicalizeQuery(u.Query())
	auth, err := SignAuthorization(ak, sk, method, host, path, canonicalQuery, now, defaultExpirationSeconds)
	if err != nil {
		return err
	}
	header.Set("Host", host)
	header.Set("Authorization", auth)
	return nil
}

func canonicalizeQuery(values url.Values) string {
	if len(values) == 0 {
		return ""
	}
	keys := make([]string, 0, len(values))
	for k := range values {
		if strings.EqualFold(k, "authorization") {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		vs := values[k]
		if len(vs) == 0 {
			parts = append(parts, uriEncode(k, true)+"=")
			continue
		}
		for _, v := range vs {
			parts = append(parts, uriEncode(k, true)+"="+uriEncode(v, true))
		}
	}
	sort.Strings(parts)
	return strings.Join(parts, "&")
}
```

Fix `canonicalURI` usage: Baidu `UriEncodeExceptSlash` means encode path but keep `/`. Prefer passing `u.Path` (decoded) then `uriEncode(path, false)`, not double-escaped `EscapedPath`. In `ApplyBCEAuth` use `path := u.Path`.

- [ ] **Step 4: Run tests 鈥?expect PASS**

```bash
go test ./relay/channel/baidu_vod_minimax/ -run "TestParseAccessKeys|TestSignAuthorization|TestApplyBCEAuth" -count=1
```

Expected: PASS

- [ ] **Step 5: Commit** (only if user asked to commit)

```bash
git add relay/channel/baidu_vod_minimax/sign.go relay/channel/baidu_vod_minimax/sign_test.go
git commit -m "feat(baidu-vod-minimax): add BCE bce-auth-v1 signer"
```

---

