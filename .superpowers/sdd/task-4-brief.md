### Task 4: Adaptor header smoke + model registry / pricing

**Files:**
- Modify: `relay/channel/baidu_vod_minimax/adaptor_test.go`
- Modify: `controller/model.go` 鈥?append `baidu_vod_minimax.ModelList` OwnedBy
- Modify: `model/pricing_default.go` 鈥?`"baidu_vod_minimax": "Baidu VOD MiniMax"`

- [ ] **Step 1: Header smoke test**

```go
func TestSetupRequestHeaderUsesBCENotBearer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/audio/speech", nil)
	c.Request.Header.Set("Content-Type", "application/json")

	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl: "https://vod.bj.baidubce.com",
			ApiKey:         "ak|sk",
		},
		RelayMode: relayconstant.RelayModeAudioSpeech,
	}
	a := &Adaptor{}
	h := make(http.Header)
	require.NoError(t, a.SetupRequestHeader(c, &h, info))
	assert.Equal(t, "vod.bj.baidubce.com", h.Get("Host"))
	assert.True(t, strings.HasPrefix(h.Get("Authorization"), "bce-auth-v1/"))
	assert.NotContains(t, h.Get("Authorization"), "Bearer")
	assert.Contains(t, h.Get("Authorization"), "/host/")
}
```

- [ ] **Step 2: Register models + pricing display**

`controller/model.go` after minimax loop:

```go
for _, modelName := range baidu_vod_minimax.ModelList {
	openAIModels = append(openAIModels, dto.OpenAIModels{
		Id: modelName, Object: "model", Created: 1626777600, OwnedBy: baidu_vod_minimax.ChannelName,
	})
}
```

`model/pricing_default.go`:

```go
"baidu_vod_minimax": "Baidu VOD MiniMax",
```

- [ ] **Step 3: Run**

```bash
go test ./relay/channel/baidu_vod_minimax/ ./controller/ -count=1 -run 'TestSetupRequestHeader|TestConvert|TestHandle|TestSign'
```

- [ ] **Step 4: Commit** (if requested)

---

