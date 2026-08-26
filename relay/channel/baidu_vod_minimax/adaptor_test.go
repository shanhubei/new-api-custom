package baidu_vod_minimax

import (
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"

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
