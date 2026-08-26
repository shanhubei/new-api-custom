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
	vs := body["voice_setting"].(map[string]any)
	assert.Equal(t, "Boyan_new_hd", vs["voice_id"])
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
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/audio/speech", nil)
	body := `{"data":{"audio":"https://example.com/a.mp3","status":1},"extra_info":{"usage_characters":20},"base_resp":{"status_code":0,"status_msg":"OK"}}`
	resp := &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}
	info := &relaycommon.RelayInfo{}
	usage, err := HandleTTSResponse(c, resp, info)
	require.Nil(t, err)
	u := usage.(*dto.Usage)
	assert.Equal(t, 20, u.TotalTokens)
	assert.Equal(t, http.StatusFound, w.Code)
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
