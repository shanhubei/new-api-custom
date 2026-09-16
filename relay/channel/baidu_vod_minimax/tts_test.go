package baidu_vod_minimax

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertOpenAIAudioToTTSRequestMapsFields(t *testing.T) {
	speed := 1.2
	req := dto.AudioRequest{
		Model:          "speech-2.8-hd",
		Input:          "你好",
		Voice:          "Boyan_new_hd",
		Speed:          &speed,
		ResponseFormat: "mp3",
	}
	info := &relaycommon.RelayInfo{OriginModelName: "speech-2.8-hd"}
	raw, outFmt, err := ConvertOpenAIAudioToTTSRequest(info, req)
	require.NoError(t, err)
	assert.Equal(t, "url", outFmt)

	var body map[string]any
	require.NoError(t, common.Unmarshal(raw, &body))
	assert.Equal(t, "speech-2.8-hd", body["model"])
	assert.Equal(t, "你好", body["text"])
	assert.Equal(t, "url", body["output_format"])
	assert.Equal(t, "Boyan_new_hd", body["voiceId"])
	vs := body["voice_setting"].(map[string]any)
	assert.Equal(t, "Boyan_new_hd", vs["voice_id"])
}

func TestConvertOpenAIAudioToTTSRequestMetadataDoesNotClearVoice(t *testing.T) {
	req := dto.AudioRequest{
		Model:          "speech-2.8-hd",
		Input:          "你好",
		Voice:          "Boyan_new_hd",
		ResponseFormat: "mp3",
		Metadata:       []byte(`{"voice_setting":{"emotion":"calm","vol":1.0}}`),
	}
	info := &relaycommon.RelayInfo{OriginModelName: "speech-2.8-hd"}
	raw, _, err := ConvertOpenAIAudioToTTSRequest(info, req)
	require.NoError(t, err)

	var body map[string]any
	require.NoError(t, common.Unmarshal(raw, &body))
	assert.Equal(t, "Boyan_new_hd", body["voiceId"])
	vs := body["voice_setting"].(map[string]any)
	assert.Equal(t, "Boyan_new_hd", vs["voice_id"])
	assert.Equal(t, "calm", vs["emotion"])
}

func TestConvertOpenAIAudioToTTSRequestPrefersUpstreamModelName(t *testing.T) {
	req := dto.AudioRequest{
		Model:          "baidu-speech-2.8-hd",
		Input:          "你好",
		Voice:          "Boyan_new_hd",
		ResponseFormat: "mp3",
	}
	info := &relaycommon.RelayInfo{
		OriginModelName: "baidu-speech-2.8-hd",
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "speech-2.8-hd",
		},
	}
	raw, _, err := ConvertOpenAIAudioToTTSRequest(info, req)
	require.NoError(t, err)

	var body map[string]any
	require.NoError(t, common.Unmarshal(raw, &body))
	assert.Equal(t, "speech-2.8-hd", body["model"])
}

func TestHandleTTSResponseURL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/audio/speech", nil)
	body := `{"data":{"audio":"https://example.com/a.mp3","status":1},"extra_info":{"usage_characters":20},"base_resp":{"status_code":0,"status_msg":"OK"}}`
	resp := &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}
	info := &relaycommon.RelayInfo{}
	usage, err := HandleTTSResponse(c, resp, info)
	require.Nil(t, err)
	u := usage.(*dto.Usage)
	assert.Equal(t, 20, u.TotalTokens)
	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "https://example.com/a.mp3", w.Header().Get("Location"))
}

func TestHandleTTSResponseErrorStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"data":{"audio":"","status":0},"extra_info":{"usage_characters":0},"base_resp":{"status_code":1001,"status_msg":"invalid request"}}`
	resp := &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}
	info := &relaycommon.RelayInfo{}
	usage, err := HandleTTSResponse(c, resp, info)
	require.NotNil(t, err)
	assert.Nil(t, usage)
}

func TestHandleTTSResponseCamelCase(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/baidu-vod/tts", nil)
	c.Set("baidu_vod_tts_json_url", true)
	body := `{"data":{"audio":"https://example.com/a.mp3","status":1},"extraInfo":{"usageCharacters":18},"baseResp":{"statusCode":0,"statusMsg":"OK"}}`
	resp := &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}
	info := &relaycommon.RelayInfo{}
	usage, err := HandleTTSResponse(c, resp, info)
	require.Nil(t, err)
	u := usage.(*dto.Usage)
	assert.Equal(t, 18, u.TotalTokens)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, body, w.Body.String())
}

func TestHandleTTSResponseEmptyAudioIncludesBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"data":{"audio":"","status":0},"base_resp":{"status_code":0,"status_msg":"OK"},"note":"empty-audio-fixture"}`
	resp := &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}
	info := &relaycommon.RelayInfo{}
	usage, err := HandleTTSResponse(c, resp, info)
	require.NotNil(t, err)
	assert.Nil(t, usage)
	assert.Contains(t, err.Error(), "empty-audio-fixture")
}

func TestHandleTTSResponseTopLevelURL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/baidu-vod/tts", nil)
	c.Set("baidu_vod_tts_json_url", true)
	body := `{"url":"https://bce-multimedia.cdn.bcebos.com/tmp/minimax/demo.mp3"}`
	resp := &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}
	info := &relaycommon.RelayInfo{}
	usage, err := HandleTTSResponse(c, resp, info)
	require.Nil(t, err)
	require.NotNil(t, usage)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, body, w.Body.String())
}

func TestHandleTTSResponsePassthroughOfficialBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/baidu-vod/tts", nil)
	c.Set("baidu_vod_tts_json_url", true)
	body := `{"data":{"audio":"https://bce-multimedia.cdn.bcebos.com/tmp/minimax/xxx.mp3","status":1},"trace_id":"trace-abc123","extra_info":{"usage_characters":20,"audio_length":3200,"audio_sample_rate":32000,"audio_size":51200,"bitrate":128,"audio_format":"mp3","audio_channel":1,"word_count":18},"base_resp":{"status_code":0,"status_msg":"OK"},"credits":12}`
	resp := &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}
	info := &relaycommon.RelayInfo{}
	usage, err := HandleTTSResponse(c, resp, info)
	require.Nil(t, err)
	u := usage.(*dto.Usage)
	assert.Equal(t, 12, u.TotalTokens)
	assert.True(t, info.PriceData.UsePrice)
	assert.InDelta(t, 1.2, info.PriceData.ModelPrice, 1e-9)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, body, w.Body.String())
}

func TestHandleTTSResponseUsageCharactersFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/baidu-vod/tts", nil)
	c.Set("baidu_vod_tts_json_url", true)
	body := `{"data":{"audio":"https://example.com/a.mp3","status":1},"extra_info":{"usage_characters":20},"base_resp":{"status_code":0,"status_msg":"OK"}}`
	resp := &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}
	info := &relaycommon.RelayInfo{}
	usage, err := HandleTTSResponse(c, resp, info)
	require.Nil(t, err)
	u := usage.(*dto.Usage)
	assert.Equal(t, 20, u.PromptTokens)
	assert.Equal(t, 20, u.TotalTokens)
	assert.False(t, info.PriceData.UsePrice)
}

func TestHandleTTSResponseHexBinary(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/audio/speech", nil)
	// "ID3" mp3-like header bytes as hex
	body := `{"data":{"audio":"494433","status":1},"extra_info":{"usage_characters":3},"base_resp":{"status_code":0,"status_msg":"OK"}}`
	resp := &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}
	info := &relaycommon.RelayInfo{}
	usage, err := HandleTTSResponse(c, resp, info)
	require.Nil(t, err)
	require.NotNil(t, usage)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "audio/mpeg", w.Header().Get("Content-Type"))
	assert.Equal(t, []byte{0x49, 0x44, 0x33}, w.Body.Bytes())
}
