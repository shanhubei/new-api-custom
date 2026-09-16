# Task 3 Report: TTS Convert + Response Handler

## Status

**DONE**

## Commits

| SHA | Subject |
|-----|---------|
| `aad9917c` | feat(baidu-vod-minimax): implement TTS convert and response |

## Files Created

- `relay/channel/baidu_vod_minimax/tts.go` — MiniMax-compatible TTS structs, `ConvertOpenAIAudioToTTSRequest`, `HandleTTSResponse`
- `relay/channel/baidu_vod_minimax/tts_test.go` — convert mapping, URL response, error status tests

## Files Modified

- `relay/channel/baidu_vod_minimax/adaptor.go` — wire `ConvertAudioRequest` + `DoResponse` for `RelayModeAudioSpeech`

## TDD Summary

1. **RED** — Added convert + response tests. Build failed with `undefined: ConvertOpenAIAudioToTTSRequest` / `HandleTTSResponse` (expected).
2. **GREEN** — Implemented `tts.go` (structs + convert + handler using `common.Marshal`/`Unmarshal`) and wired adaptor. All package tests PASS.

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
=== RUN   TestConvertOpenAIAudioToTTSRequestMapsFields
--- PASS: TestConvertOpenAIAudioToTTSRequestMapsFields (0.00s)
=== RUN   TestHandleTTSResponseURL
--- PASS: TestHandleTTSResponseURL (0.00s)
=== RUN   TestHandleTTSResponseErrorStatus
--- PASS: TestHandleTTSResponseErrorStatus (0.00s)
PASS
ok  	github.com/QuantumNous/new-api/relay/channel/baidu_vod_minimax	6.471s
```

## Implementation Notes

### ConvertOpenAIAudioToTTSRequest

| OpenAI field | Upstream field |
|--------------|----------------|
| `OriginModelName` | `model` |
| `input` | `text` |
| `voice` | `voice_setting.voice_id` |
| `speed` | `voice_setting.speed` |
| `response_format` | `audio_setting.format`; non-`hex` → `output_format=url` |
| `metadata` | merged into request struct via `common.Unmarshal` |

Returns `([]byte, outFmt, error)`. Adaptor sets `c.Set("response_format", outFmt)` and returns `bytes.NewReader(raw)`.

### HandleTTSResponse

- `base_resp.status_code != 0` → error
- empty `data.audio` → error
- `data.audio` http(s) URL → `302` redirect
- otherwise hex-decode → `audio/mpeg` bytes
- usage: `extra_info.usage_characters` → `TotalTokens`

### Self-Review

| Check | Result |
|-------|--------|
| Uses `common.Marshal`/`Unmarshal` (not direct encoding/json calls) | ✅ |
| Types duplicated in package (no import of minimax package) | ✅ |
| `ConvertAudioRequest` only speech mode | ✅ |
| `DoResponse` routes speech → `HandleTTSResponse` | ✅ |
| testify require/assert | ✅ |
| Field tags match MiniMax TTS | ✅ |

## Concerns

- Brief's `ChannelMeta{OriginModelName}` is invalid; `OriginModelName` lives on `RelayInfo` — test corrected accordingly.
- Gin `Redirect` with POST does not flush status to `httptest.ResponseRecorder` (no body write); URL test uses GET so `w.Code == 302` is observable. Production speech requests are POST but clients still receive Location via gin's writer status.
- Hex path and metadata-merge not separately unit-tested beyond convert field mapping (aligned with MiniMax behavior; optional follow-up).
- Non-speech `DoResponse` still returns not-implemented (out of Task 3 scope).

## Next Task

Task 4 (if any): end-to-end wiring / frontend channel picker / integration polish.
