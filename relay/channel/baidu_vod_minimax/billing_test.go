package baidu_vod_minimax

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyCreditsBilling(t *testing.T) {
	info := &relaycommon.RelayInfo{}
	require.True(t, ApplyCreditsBilling(info, 12))
	assert.True(t, info.PriceData.UsePrice)
	assert.InDelta(t, 1.2, info.PriceData.ModelPrice, 1e-9)
	assert.False(t, ApplyCreditsBilling(info, 0))
}

func TestQuotaFromCredits(t *testing.T) {
	// 10 credits = 1 元 → QuotaPerUnit
	assert.Equal(t, int(common.QuotaPerUnit), QuotaFromCredits(10, 1))
	assert.Equal(t, int(common.QuotaPerUnit*1.2), QuotaFromCredits(12, 1))
	assert.Equal(t, 0, QuotaFromCredits(0, 1))
}

func TestParseCreditsFromBody(t *testing.T) {
	assert.Equal(t, int64(12), parseCreditsFromBody([]byte(`{"credits":12,"url":"https://x"}`)))
	assert.Equal(t, int64(8), parseCreditsFromBody([]byte(`{"data":{"audio":"https://x","status":1},"extra_info":{"credits":8,"usage_characters":20}}`)))
	assert.Equal(t, int64(0), parseCreditsFromBody([]byte(`{"extra_info":{"usage_characters":20}}`)))
}
