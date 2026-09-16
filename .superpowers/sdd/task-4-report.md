# Task 4 Report: Adaptor Header Smoke + Model Registry / Pricing

## Status

**DONE**

## Commits

| SHA | Subject |
|-----|---------|
| `21f97bd3` | feat(baidu-vod-minimax): register models and header smoke test |

## Files Modified

- `relay/channel/baidu_vod_minimax/adaptor_test.go` — `TestSetupRequestHeaderUsesBCENotBearer` smoke test
- `controller/model.go` — register `baidu_vod_minimax.ModelList` with `OwnedBy: baidu_vod_minimax.ChannelName`
- `model/pricing_default.go` — `"baidu_vod_minimax": "Baidu VOD MiniMax"` vendor rule

## Test Results

```
=== RUN   TestGetRequestURL
--- PASS: TestGetRequestURL (0.00s)
=== RUN   TestSetupRequestHeaderUsesBCENotBearer
--- PASS: TestSetupRequestHeaderUsesBCENotBearer (0.00s)
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
ok  	github.com/QuantumNous/new-api/relay/channel/baidu_vod_minimax	6.624s
```

## Implementation Notes

### TestSetupRequestHeaderUsesBCENotBearer

Asserts `Adaptor.SetupRequestHeader` sets BCE auth headers (not Bearer):

- `Host: vod.bj.baidubce.com`
- `Authorization` prefix `bce-auth-v1/`
- contains `/host/` signed header segment
- does not contain `Bearer`

### Model registry

Eight speech models from `baidu_vod_minimax.ModelList` appended to `openAIModels` in `controller/model.go` after the MiniMax loop, with `OwnedBy: "baidu_vod_minimax"`.

### Pricing vendor mapping

Added `"baidu_vod_minimax": "Baidu VOD MiniMax"` to `defaultVendorRules` for default vendor assignment during pricing init.

## Self-Review

| Check | Result |
|-------|--------|
| Header smoke uses BCE not Bearer | ✅ |
| All 8 models registered in controller | ✅ |
| Pricing vendor rule added | ✅ |
| Package tests pass | ✅ |
