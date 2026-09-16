package baidu_vod_vidu

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

// AdjustBillingOnComplete settles lip-sync by upstream credits.
// 1 credit = 1 RMB → quota = credits × QuotaPerUnit × GroupRatio.
// Returns 0 to keep pre-charge when credits missing or action is not lip-sync.
func (a *TaskAdaptor) AdjustBillingOnComplete(task *model.Task, _ *relaycommon.TaskInfo) int {
	if task == nil || task.Action != constant.TaskActionLipSync {
		return 0
	}

	credits := parseTaskCredits(task.Data)
	if credits <= 0 {
		credits = task.PrivateData.UpstreamCredits
	}
	if credits <= 0 {
		return 0
	}

	groupRatio := 1.0
	if bc := task.PrivateData.BillingContext; bc != nil && bc.GroupRatio > 0 {
		groupRatio = bc.GroupRatio
	}

	actual, _ := common.QuotaFromFloatChecked(float64(credits) * common.QuotaPerUnit * groupRatio)
	return actual
}

func parseTaskCredits(respBody []byte) int {
	if len(respBody) == 0 {
		return 0
	}
	var resp struct {
		Credits int `json:"credits"`
	}
	if err := common.Unmarshal(respBody, &resp); err != nil {
		return 0
	}
	if resp.Credits <= 0 {
		return 0
	}
	return resp.Credits
}
