package baidu_vod_vidu

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildRequestURLImg2Video(t *testing.T) {
	adaptor := &TaskAdaptor{}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl: "http://vod.bj.baidubce.com/v3/aigc/vd",
			ChannelType:    constant.ChannelTypeBaiduVodVidu,
		},
		TaskRelayInfo: &relaycommon.TaskRelayInfo{
			Action: constant.TaskActionGenerate,
		},
	}
	adaptor.Init(info)

	url, err := adaptor.BuildRequestURL(info)
	require.NoError(t, err)
	assert.Equal(t, "http://vod.bj.baidubce.com/v3/aigc/vd/ent/v2/img2video", url)
}

func TestBuildRequestHeaderUsesBearer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adaptor := &TaskAdaptor{}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl: "http://vod.bj.baidubce.com/v3/aigc/vd",
			ChannelType:    constant.ChannelTypeBaiduVodVidu,
			ApiKey:         "bce-v3/ALTAK-test/secret",
		},
	}
	adaptor.Init(info)

	req, err := http.NewRequest(http.MethodPost, "http://example.com", nil)
	require.NoError(t, err)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	err = adaptor.BuildRequestHeader(c, req, info)
	require.NoError(t, err)
	assert.Equal(t, "Bearer bce-v3/ALTAK-test/secret", req.Header.Get("Authorization"))
	assert.NotEqual(t, "Token bce-v3/ALTAK-test/secret", req.Header.Get("Authorization"))
}

func TestConvertToRequestPayloadIncludesModerationFromMetadata(t *testing.T) {
	adaptor := &TaskAdaptor{}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       constant.ChannelTypeBaiduVodVidu,
			UpstreamModelName: "viduq3-pro",
		},
	}
	req := &relaycommon.TaskSubmitReq{
		Prompt:   "The astronaut waved",
		Images:   []string{"https://example.com/frame.png"},
		Duration: 5,
		Size:     "720p",
		Metadata: map[string]any{
			"moderation":         "disabled",
			"audio":              false,
			"off_peak":           false,
			"movement_amplitude": "auto",
		},
	}

	body, err := adaptor.convertToRequestPayload(req, info)
	require.NoError(t, err)
	require.NotNil(t, body.Moderation)
	assert.Equal(t, "disabled", *body.Moderation)
	require.NotNil(t, body.Audio)
	assert.Equal(t, false, *body.Audio)
	require.NotNil(t, body.OffPeak)
	assert.Equal(t, false, *body.OffPeak)

	raw, err := common.Marshal(body)
	require.NoError(t, err)
	s := string(raw)
	assert.Contains(t, s, `"moderation":"disabled"`)
	assert.Contains(t, s, `"audio":false`)
	assert.Contains(t, s, `"off_peak":false`)
}

func TestGetChannelName(t *testing.T) {
	adaptor := &TaskAdaptor{}
	assert.Equal(t, "baidu_vod_vidu", adaptor.GetChannelName())
}

func TestBuildRequestURLReference2Image(t *testing.T) {
	adaptor := &TaskAdaptor{}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl: "http://vod.bj.baidubce.com/v3/aigc/vd",
			ChannelType:    constant.ChannelTypeBaiduVodVidu,
		},
		TaskRelayInfo: &relaycommon.TaskRelayInfo{
			Action: constant.TaskActionReference2Image,
		},
	}
	adaptor.Init(info)

	url, err := adaptor.BuildRequestURL(info)
	require.NoError(t, err)
	assert.Equal(t, "http://vod.bj.baidubce.com/v3/aigc/vd/ent/v2/reference2image", url)
}

func TestParseImageRequestPayloadOmitsVideoDefaults(t *testing.T) {
	raw := []byte(`{"model":"viduq2","prompt":"生成一个打游戏的画面"}`)
	body, err := parseImageRequestPayload(raw, "viduq2")
	require.NoError(t, err)
	assert.Equal(t, "viduq2", body.Model)
	assert.Equal(t, "生成一个打游戏的画面", body.Prompt)

	s, err := common.Marshal(body)
	require.NoError(t, err)
	out := string(s)
	assert.NotContains(t, out, `"duration"`)
	assert.NotContains(t, out, `"movement_amplitude"`)
	assert.NotContains(t, out, `"bgm"`)
	assert.NotContains(t, out, `"moderation"`)
}

func TestParseImageRequestPayloadKeepsOfficialOptionalFields(t *testing.T) {
	raw := []byte(`{
		"model":"viduq2",
		"prompt":"hello",
		"images":["https://example.com/a.png"],
		"seed":42,
		"aspect_ratio":"16:9",
		"resolution":"2K"
	}`)
	body, err := parseImageRequestPayload(raw, "viduq2")
	require.NoError(t, err)
	require.NotNil(t, body.Seed)
	assert.Equal(t, 42, *body.Seed)
	require.NotNil(t, body.AspectRatio)
	assert.Equal(t, "16:9", *body.AspectRatio)
	require.NotNil(t, body.Resolution)
	assert.Equal(t, "2K", *body.Resolution)

	s, err := common.Marshal(body)
	require.NoError(t, err)
	out := string(s)
	assert.Contains(t, out, `"aspect_ratio":"16:9"`)
	assert.Contains(t, out, `"resolution":"2K"`)
	assert.Contains(t, out, `"seed":42`)
}

func TestValidateRequestSetsReference2ImageAction(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adaptor := &TaskAdaptor{}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := bytes.NewBufferString(`{"model":"viduq2","prompt":"生成一个打游戏的画面"}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/async/images", body)
	c.Request.Header.Set("Content-Type", "application/json")

	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       constant.ChannelTypeBaiduVodVidu,
			UpstreamModelName: "viduq2",
		},
		TaskRelayInfo:   &relaycommon.TaskRelayInfo{},
		OriginModelName: "viduq2",
	}
	adaptor.Init(info)

	taskErr := adaptor.ValidateRequestAndSetAction(c, info)
	require.Nil(t, taskErr)
	assert.Equal(t, constant.TaskActionReference2Image, info.Action)
}

func TestValidateViduq1WithoutImagesFails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adaptor := &TaskAdaptor{}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := bytes.NewBufferString(`{"model":"viduq1","prompt":"no image"}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/async/images", body)
	c.Request.Header.Set("Content-Type", "application/json")

	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       constant.ChannelTypeBaiduVodVidu,
			UpstreamModelName: "viduq1",
		},
		TaskRelayInfo:   &relaycommon.TaskRelayInfo{},
		OriginModelName: "viduq1",
	}
	adaptor.Init(info)

	taskErr := adaptor.ValidateRequestAndSetAction(c, info)
	require.NotNil(t, taskErr)
	assert.Equal(t, http.StatusBadRequest, taskErr.StatusCode)
}

