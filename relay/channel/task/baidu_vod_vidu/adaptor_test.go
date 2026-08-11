package baidu_vod_vidu

import (
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
