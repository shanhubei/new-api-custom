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
	host = strings.TrimSpace(host)
	if canonicalURI == "" {
		canonicalURI = "/"
	}
	if !strings.HasPrefix(canonicalURI, "/") {
		canonicalURI = "/" + canonicalURI
	}
	timestamp := now.UTC().Format("2006-01-02T15:04:05Z")
	authPrefix := fmt.Sprintf("bce-auth-v1/%s/%s/%d", ak, timestamp, expirationSec)
	canonicalURIEnc := uriEncode(canonicalURI, false)
	canonicalHeaders := "host:" + uriEncode(host, true)
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
	host := strings.TrimSpace(u.Host)
	if host == "" {
		return fmt.Errorf("request url missing host")
	}
	path := u.Path
	if path == "" {
		path = "/"
	}
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
