package ali

import (
	"math"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

// wan27R2VMaxInputBillableSeconds is Aliyun wan2.7 r2v's input-video billing cap
// across reference videos (official: total input billable seconds capped at 5).
const wan27R2VMaxInputBillableSeconds = 5

// wan27VideoEditMaxSourceSeconds is the max source/output length for videoedit.
const wan27VideoEditMaxSourceSeconds = 10

// estimateBillableSeconds returns the conservative pre-charge seconds multiplier.
func estimateBillableSeconds(aliReq *AliVideoRequest) int {
	if aliReq == nil || aliReq.Parameters == nil {
		return 5
	}
	out := aliReq.Parameters.Duration
	switch {
	case isWan27VideoEditModel(aliReq.Model):
		// Official bill ≈ input + output. When duration=0, output follows source (≤10s).
		if out <= 0 {
			out = wan27VideoEditMaxSourceSeconds
		}
		return min(out*2, relaycommon.MaxTaskDurationSeconds)
	case isWan27R2VModel(aliReq.Model):
		if out <= 0 {
			out = 5
		}
		return min(out+wan27R2VMaxInputBillableSeconds, relaycommon.MaxTaskDurationSeconds)
	default:
		if out <= 0 {
			out = 5
		}
		return min(out, relaycommon.MaxTaskDurationSeconds)
	}
}

// AdjustBillingOnComplete recalculates quota from upstream usage for input+output models.
// Returns 0 to keep the pre-charged amount.
func (a *TaskAdaptor) AdjustBillingOnComplete(task *model.Task, _ *relaycommon.TaskInfo) int {
	if task == nil {
		return 0
	}

	modelName := task.Properties.OriginModelName
	if task.PrivateData.BillingContext != nil && task.PrivateData.BillingContext.OriginModelName != "" {
		modelName = task.PrivateData.BillingContext.OriginModelName
	}
	if !isWan27InputVideoBillingModel(modelName) {
		return 0
	}

	seconds := parseAliUsageBillableSeconds(task.Data)
	if seconds <= 0 {
		return 0
	}

	bc := task.PrivateData.BillingContext
	if bc == nil || bc.ModelPrice <= 0 {
		return 0
	}

	otherMultiplier := 1.0
	for key, ratio := range bc.OtherRatios {
		if key == "seconds" {
			continue
		}
		if ratio != 1.0 && ratio > 0 {
			otherMultiplier *= ratio
		}
	}

	actual, _ := common.QuotaFromFloatChecked(
		bc.ModelPrice * common.QuotaPerUnit * bc.GroupRatio * float64(seconds) * otherMultiplier,
	)
	return actual
}

func parseAliUsageBillableSeconds(respBody []byte) int {
	if len(respBody) == 0 {
		return 0
	}
	var aliResp AliVideoResponse
	if err := common.Unmarshal(respBody, &aliResp); err != nil {
		return 0
	}
	if aliResp.Usage == nil {
		return 0
	}
	seconds := aliResp.Usage.Duration
	if seconds <= 0 && (aliResp.Usage.InputVideoDuration > 0 || aliResp.Usage.OutputVideoDuration > 0) {
		seconds = aliResp.Usage.InputVideoDuration + aliResp.Usage.OutputVideoDuration
	}
	if seconds <= 0 {
		return 0
	}
	billed := int(math.Ceil(seconds))
	if billed < 1 {
		return 0
	}
	return min(billed, relaycommon.MaxTaskDurationSeconds)
}
