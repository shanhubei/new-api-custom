package baidu_vod_minimax

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetRequestURL(t *testing.T) {
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl: "https://vod.bj.baidubce.com",
		},
	}
	u, err := GetRequestURL(info)
	require.NoError(t, err)
	assert.Equal(t, "https://vod.bj.baidubce.com/v2/tts", u)
}

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
