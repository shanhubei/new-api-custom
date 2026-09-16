package baidu_vod_minimax

import (
	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

// creditsPerYuan: 与百度 VOD Vidu 一致，1 元人民币 = 10 上游积分 → 1 积分 = 0.1 元。
const creditsPerYuan = 10

// ApplyCreditsBilling switches settlement to per-call RMB price derived from upstream credits.
// quota = (credits / 10) × QuotaPerUnit × GroupRatio (via UsePrice + ModelPrice).
func ApplyCreditsBilling(info *relaycommon.RelayInfo, credits int64) bool {
	if info == nil || credits <= 0 {
		return false
	}
	info.PriceData.UsePrice = true
	info.PriceData.ModelPrice = float64(credits) / float64(creditsPerYuan)
	return true
}

func QuotaFromCredits(credits int64, groupRatio float64) int {
	if credits <= 0 {
		return 0
	}
	if groupRatio <= 0 {
		groupRatio = 1
	}
	q, _ := common.QuotaFromFloatChecked(
		float64(credits) / float64(creditsPerYuan) * common.QuotaPerUnit * groupRatio,
	)
	return q
}

func parseCreditsFromBody(body []byte) int64 {
	if len(body) == 0 {
		return 0
	}
	var raw map[string]any
	if err := common.Unmarshal(body, &raw); err != nil {
		return 0
	}
	if c := asPositiveInt64(raw["credits"]); c > 0 {
		return c
	}
	if data, ok := raw["data"].(map[string]any); ok {
		if c := asPositiveInt64(data["credits"]); c > 0 {
			return c
		}
	}
	if extra, ok := raw["extra_info"].(map[string]any); ok {
		if c := asPositiveInt64(extra["credits"]); c > 0 {
			return c
		}
	}
	if extra, ok := raw["extraInfo"].(map[string]any); ok {
		if c := asPositiveInt64(extra["credits"]); c > 0 {
			return c
		}
	}
	return 0
}

func asPositiveInt64(v any) int64 {
	switch n := v.(type) {
	case float64:
		if n > 0 {
			return int64(n)
		}
	case int64:
		if n > 0 {
			return n
		}
	case int:
		if n > 0 {
			return int64(n)
		}
	}
	return 0
}
