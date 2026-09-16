# Task 1 Report: BCE bce-auth-v1 Signer

## Status

**DONE**

## Commits

| SHA | Subject |
|-----|---------|
| `ff76b2d4` | feat(baidu-vod-minimax): add BCE bce-auth-v1 signer |

## Files Created

- `relay/channel/baidu_vod_minimax/sign.go`
- `relay/channel/baidu_vod_minimax/sign_test.go`

## TDD Summary

1. **RED** — Created `sign_test.go` with four tests from the task brief. Ran `go test ./relay/channel/baidu_vod_minimax/ -run "TestParseAccessKeys|TestSignAuthorization|TestApplyBCEAuth" -count=1`. Result: build failed (undefined symbols) as expected.
2. **GREEN** — Implemented `sign.go` per Baidu BCE v1 spec. Re-ran same test command. Result: all 4 tests PASS in 1.8s.

## Test Results

```
=== RUN   TestParseAccessKeys
--- PASS: TestParseAccessKeys (0.00s)
=== RUN   TestSignAuthorizationContainsHostSignedHeader
--- PASS: TestSignAuthorizationContainsHostSignedHeader (0.00s)
=== RUN   TestSignAuthorizationDeterministic
--- PASS: TestSignAuthorizationDeterministic (0.00s)
=== RUN   TestApplyBCEAuthSetsHostAndAuthorization
--- PASS: TestApplyBCEAuthSetsHostAndAuthorization (0.00s)
PASS
ok  	github.com/QuantumNous/new-api/relay/channel/baidu_vod_minimax	1.827s
```

## Implementation Notes

### Public API

| Function | Purpose |
|----------|---------|
| `ParseAccessKeys(apiKey string) (ak, sk string, err error)` | Splits `accessKey\|secretKey` format; trims whitespace; rejects malformed input |
| `SignAuthorization(ak, sk, method, host, canonicalURI, canonicalQuery string, now time.Time, expirationSec int) (authorization string, err error)` | Builds `bce-auth-v1/{ak}/{timestamp}/{expiration}/host/{signature}` |
| `ApplyBCEAuth(header *http.Header, apiKey, method, requestURL string, now time.Time) error` | Parses URL, signs request, sets `Host` and `Authorization` headers |

### Key Design Decisions

- **Expiration**: Fixed at `1800` seconds via `defaultExpirationSeconds` constant; used in `ApplyBCEAuth` and as fallback when `expirationSec <= 0` in `SignAuthorization`.
- **Signed headers**: Only `host` is signed (`signedHeaders` = `"host"`).
- **Canonical headers**: `host:{UriEncode(host)}` with `encodeSlash=true`.
- **Canonical URI**: `UriEncodeExceptSlash` — `uriEncode(path, false)` preserves `/`.
- **ApplyBCEAuth path fix**: Uses `u.Path` (decoded path), **not** `u.EscapedPath`, to avoid double-encoding per task brief.
- **Query canonicalization**: Sorted keys, skip `authorization` (case-insensitive), UriEncode both key and value, empty values produce `key=`.

### Self-Review

| Check | Result |
|-------|--------|
| No real AK/SK in code | ✅ Test keys are placeholders (`ak-test`, `sk-test`, `ak\|sk`) |
| Uses `u.Path` not `EscapedPath` | ✅ Line 95 in `sign.go` |
| `canonicalHeaders` = `host:` + UriEncode(host) | ✅ |
| signedHeaders exactly `"host"` | ✅ Auth string format `{prefix}/host/{sig}` |
| expiration fixed at 1800 | ✅ |
| testify require/assert in tests | ✅ |
| No channel constants wired | ✅ Only signer package files created |
| Linter clean | ✅ No diagnostics |

## Concerns

None. Implementation matches task brief verbatim except for the intentional `u.Path` fix and removal of a duplicate `canonicalHeaders` assignment present in the brief's draft code.

## Next Task

Task 2 should wire channel type constants and integrate this signer into the Baidu VOD MiniMax adaptor.

---

## Review Fix (2026-08-26)

### Status

**DONE** — Task 1 review findings addressed.

### Commit

| SHA | Subject |
|-----|---------|
| `594502be` | fix(baidu-vod-minimax): golden-vector test and trim host for BCE sign |

### Changes

1. **Golden-vector test** — Added `TestSignAuthorizationGoldenVector` with hardcoded expected Authorization string and independent HMAC-SHA256 recomputation in the test.
2. **Host TrimSpace** — `SignAuthorization` and `ApplyBCEAuth` now trim host whitespace before canonical headers and `Host` header.
3. **Trim regression test** — Added `TestSignAuthorizationTrimsHost` asserting padded host produces the same golden Authorization string.

### Test Results

```
=== RUN   TestParseAccessKeys
--- PASS: TestParseAccessKeys (0.00s)
=== RUN   TestSignAuthorizationGoldenVector
--- PASS: TestSignAuthorizationGoldenVector (0.00s)
=== RUN   TestSignAuthorizationTrimsHost
--- PASS: TestSignAuthorizationTrimsHost (0.00s)
=== RUN   TestSignAuthorizationContainsHostSignedHeader
--- PASS: TestSignAuthorizationContainsHostSignedHeader (0.00s)
=== RUN   TestSignAuthorizationDeterministic
--- PASS: TestSignAuthorizationDeterministic (0.00s)
=== RUN   TestApplyBCEAuthSetsHostAndAuthorization
--- PASS: TestApplyBCEAuthSetsHostAndAuthorization (0.00s)
PASS
ok  	github.com/QuantumNous/new-api/relay/channel/baidu_vod_minimax	2.027s
```

Golden Authorization string:

`bce-auth-v1/ak-test/2026-06-12T02:45:13Z/1800/host/b6952868b8ae3da6a73cd732e90d620f23f6ae3ecce40832c97fdcd729f8902f`
