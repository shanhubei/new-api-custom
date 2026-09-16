# Task 6 Report: Final Verification — Baidu VOD MiniMax TTS

## Status

**DONE** — all automated checks green; no code changes required.

## Branch

`feat/baidu-vod-minimax-tts`

## Step 1: Full Package Tests

```bash
go test ./relay/channel/baidu_vod_minimax/ -count=1 -v
```

**Result:** PASS — **11/11** tests in 0.198s

| Test | Result |
|------|--------|
| TestGetRequestURL | PASS |
| TestSetupRequestHeaderUsesBCENotBearer | PASS |
| TestParseAccessKeys | PASS |
| TestSignAuthorizationGoldenVector | PASS |
| TestSignAuthorizationTrimsHost | PASS |
| TestSignAuthorizationContainsHostSignedHeader | PASS |
| TestSignAuthorizationDeterministic | PASS |
| TestApplyBCEAuthSetsHostAndAuthorization | PASS |
| TestConvertOpenAIAudioToTTSRequestMapsFields | PASS |
| TestHandleTTSResponseURL | PASS |
| TestHandleTTSResponseErrorStatus | PASS |

## Step 2: ChannelBaseURLs Length Check

Programmatic check (`go run` against `constant` package):

```
BaiduVodMinimax 60
BaiduVodVidu 59
ChannelTypeDummy 60
len(ChannelBaseURLs) 61
ChannelBaseURLs[59] http://vod.bj.baidubce.com/v3/aigc/vd
ChannelBaseURLs[60] https://vod.bj.baidubce.com
len == ChannelTypeDummy + 1  true
```

**Interpretation:** After adding channel type 60, `ChannelBaseURLs` has **61 entries** (indices 0–60). `ChannelTypeBaiduVodMinimax = 60`; `ChannelTypeDummy` shares numeric value **60** (count sentinel = last channel index). The project invariant is `len(ChannelBaseURLs) == ChannelTypeDummy + 1` (equivalently `len == ChannelTypeBaiduVodMinimax + 1`). Index 60 holds the expected default base `https://vod.bj.baidubce.com`.

## Step 3: Manual Checklist (for PR / staging)

- [ ] **Admin — create channel:** Type **Baidu VOD MiniMax** (60); Base empty or `https://vod.bj.baidubce.com`; Key `AK|SK`.
- [ ] **Model routing:** Enable `speech-2.8-hd` on that channel only; avoid same group as official MiniMax with the same model name.
- [ ] **TTS call:** `POST /v1/audio/speech` with that model → audio URL or bytes returned.
- [ ] **Regression:** Official MiniMax (35) and Baidu VOD Vidu (59) still work unchanged.

## Spec Coverage (Tasks 1–6)

| Requirement | Verified by |
|-------------|-------------|
| BCE `bce-auth-v1` + `host` + `AK\|SK` | Task 1 tests + Task 4 header smoke |
| Channel type 60 + default Base | Task 2 + length check above |
| `/v2/tts` URL | TestGetRequestURL |
| OpenAI speech → MiniMax body | TestConvertOpenAIAudioToTTSRequestMapsFields |
| Response url/hex + usage_characters | TestHandleTTSResponseURL |
| Model list `speech-*` | Task 4 |
| Frontend selectable (type 60) | Task 5 |
| No change to MiniMax / Vidu packages | Scoped commits; no edits under `relay/channel/minimax` or baidu vod vidu adaptor |

**Pinned decisions:** `expirationSeconds = 1800`; signedHeaders = `host` only; phase-1 TTS only.

## Commits (feature branch, no Task 6 commit)

Task 6 is verification-only; branch already contains Tasks 1–5 commits. No empty commit created.

## Final review fix

**Status:** DONE — Important finding fixed (TTS body `model` now prefers channel-mapped upstream name).

**Commit:** `12cfa579` — `fix(baidu-vod-minimax): send upstream mapped model name in TTS body`

**Change:** `ConvertOpenAIAudioToTTSRequest` sets `model` from `ChannelMeta.UpstreamModelName` when non-empty, else `OriginModelName`.

**Tests:** `go test ./relay/channel/baidu_vod_minimax/ -count=1` — PASS (12/12, includes `TestConvertOpenAIAudioToTTSRequestPrefersUpstreamModelName`).
