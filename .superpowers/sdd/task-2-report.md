# Task 2 Report: Channel / APIType Wiring

## Status

**DONE**

## Commits

| SHA | Subject |
|-----|---------|
| `9cba93a3` | feat(baidu-vod-minimax): wire channel type 60 and adaptor stub |

## Files Modified

- `constant/channel.go` — `ChannelTypeBaiduVodMinimax = 60`, base URL, display name
- `constant/api_type.go` — `APITypeBaiduVodMinimax` before Dummy
- `common/api_type.go` — `ChannelType2APIType` case
- `relay/relay_adaptor.go` — import + `GetAdaptor` case

## Files Created

- `relay/channel/baidu_vod_minimax/constants.go` — model list + channel name
- `relay/channel/baidu_vod_minimax/adaptor.go` — compiling stub (`GetRequestURL`, `SetupRequestHeader` with `ApplyBCEAuth`)
- `relay/channel/baidu_vod_minimax/adaptor_test.go` — `TestGetRequestURL`

## TDD Summary

1. **RED** — Added `TestGetRequestURL`. Ran `go test ./relay/channel/baidu_vod_minimax/ -count=1 -run TestGetRequestURL`. Result: build failed (`undefined: GetRequestURL`) as expected.
2. **GREEN** — Wired constants + stub adaptor. Re-ran package tests. Result: all PASS.

## Test Results

```
=== RUN   TestGetRequestURL
--- PASS: TestGetRequestURL (0.00s)
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
ok  	github.com/QuantumNous/new-api/relay/channel/baidu_vod_minimax	0.191s
```

Compile check (no tests):

```
ok  	github.com/QuantumNous/new-api/relay/channel/baidu_vod_minimax	[no tests to run]
ok  	github.com/QuantumNous/new-api/common	[no tests to run]
ok  	github.com/QuantumNous/new-api/relay	[no tests to run]
```

Slice length verified: `ChannelTypeBaiduVodMinimax == 60`, `ChannelTypeDummy == 60`, `len(ChannelBaseURLs) == 61`, `ChannelBaseURLs[60] == "https://vod.bj.baidubce.com"`.

## Implementation Notes

### Constants

| Symbol | Value |
|--------|-------|
| `ChannelTypeBaiduVodMinimax` | `60` |
| `ChannelTypeDummy` | `60` (repeats previous const expression; upper bound for channel iteration) |
| Default base URL | `https://vod.bj.baidubce.com` |
| Display name | `Baidu VOD MiniMax` |
| `APITypeBaiduVodMinimax` | inserted before `APITypeDummy` (iota) |

### Adaptor stub

- `GetRequestURL` → `{base}/v2/tts` (trim trailing `/`; empty base falls back to `ChannelBaseURLs[60]`)
- `SetupRequestHeader` → `SetupApiRequestHeader` + JSON Accept/Content-Type + `ApplyBCEAuth(..., POST, url, now)`
- Convert methods / `DoResponse` → `not implemented` (Task 3/4)
- `DoRequest` → `channel.DoApiRequest`
- Models: speech-2.8/2.6/02/01 hd+turbo; `ChannelName = "baidu_vod_minimax"`

### Self-Review

| Check | Result |
|-------|--------|
| Type 60 before Dummy | ✅ |
| `ChannelBaseURLs` length 61 (indices 0..60) | ✅ |
| Signer from Task 1 reused, not rewritten | ✅ |
| `GetAdaptor` returns `*baidu_vod_minimax.Adaptor` | ✅ |
| Compiles against `channel.Adaptor` | ✅ |
| No Bearer auth | ✅ |
| testify require/assert in URL test | ✅ |

## Concerns

- `ConvertAudioRequest` / `DoResponse` are stubs; TTS will not work end-to-end until Task 3/4.
- Frontend channel picker / i18n not in this task scope (backend wiring only).
- `gofmt` realigned the entire `ChannelType*` const block due to the longer new identifier name (noise in diff, intentional).

## Next Task

Task 3 should implement `ConvertAudioRequest` (MiniMax-compatible TTS body) and wire speech response handling.
