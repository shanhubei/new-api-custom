### Task 3: TTS convert + response handler

**Files:**
- Create: `relay/channel/baidu_vod_minimax/tts.go`
- Create: `relay/channel/baidu_vod_minimax/tts_test.go`
- Modify: `relay/channel/baidu_vod_minimax/adaptor.go` 鈥?`ConvertAudioRequest` / `DoResponse`

**Interfaces:**
- Produces: `ConvertOpenAIAudioToTTSRequest(info, request dto.AudioRequest) ([]byte, string /*response_format*/, error)`, `HandleTTSResponse(...)`

- [ ] **Step 1: Failing tests for convert**

```go
func TestConvertOpenAIAudioToTTSRequestMapsFields(t *testing.T) {
	speed := 1.2
	req := dto.AudioRequest{
		Model:          "speech-2.8-hd",
		Input:          "浣犲ソ",
		Voice:          "Boyan_new_hd",
		Speed:          &speed,
		ResponseFormat: "mp3",
	}
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{OriginModelName: "speech-2.8-hd"}}
	raw, outFmt, err := ConvertOpenAIAudioToTTSRequest(info, req)
	require.NoError(t, err)
	assert.Equal(t, "url", outFmt)

	var body map[string]any
	require.NoError(t, common.Unmarshal(raw, &body))
	assert.Equal(t, "speech-2.8-hd", body["model"])
	assert.Equal(t, "浣犲ソ", body["text"])
	assert.Equal(t, "url", body["output_format"])
	vs := body["voice_setting"].(map[string]any)
	assert.Equal(t, "Boyan_new_hd", vs["voice_id"])
}
```

Align convert logic with `minimax.Adaptor.ConvertAudioRequest` (including metadata merge), but:

- Use `common.Marshal` / `common.Unmarshal`
- Types can live in this package (duplicate MiniMax TTS structs 鈥?acceptable to avoid coupling; keep field tags identical to spec)

- [ ] **Step 2: Failing test for response**

```go
func TestHandleTTSResponseURL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"data":{"audio":"https://example.com/a.mp3","status":1},"extra_info":{"usage_characters":20},"base_resp":{"status_code":0,"status_msg":"OK"}}`
	resp := &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}
	info := &relaycommon.RelayInfo{}
	usage, err := HandleTTSResponse(c, resp, info)
	require.Nil(t, err)
	u := usage.(*dto.Usage)
	assert.Equal(t, 20, u.TotalTokens)
	assert.Equal(t, http.StatusFound, w.Code)
}
```

Also test `status_code != 0` returns error.

- [ ] **Step 3: Implement `tts.go` + wire adaptor**

Mirror MiniMax structs/response handling; `ConvertAudioRequest` on adaptor calls convert helper; `DoResponse` if `RelayModeAudioSpeech` 鈫?`HandleTTSResponse`.

- [ ] **Step 4: Run tests PASS**

```bash
go test ./relay/channel/baidu_vod_minimax/ -count=1
```

- [ ] **Step 5: Commit** (if requested)

```bash
git commit -am "feat(baidu-vod-minimax): implement TTS convert and response"
```

---

