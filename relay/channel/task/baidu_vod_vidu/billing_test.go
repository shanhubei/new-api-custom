package baidu_vod_vidu

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdjustBillingOnCompleteLipSyncUsesCredits(t *testing.T) {
	adaptor := &TaskAdaptor{}
	body, err := common.Marshal(map[string]any{
		"state":   "success",
		"credits": 12,
	})
	require.NoError(t, err)

	task := &model.Task{
		Action: constant.TaskActionLipSync,
		Data:   body,
		PrivateData: model.TaskPrivateData{
			BillingContext: &model.TaskBillingContext{
				GroupRatio: 1.0,
			},
		},
	}

	actual := adaptor.AdjustBillingOnComplete(task, nil)
	expected, _ := common.QuotaFromFloatChecked(12.0 / 10 * common.QuotaPerUnit * 1.0)
	assert.Equal(t, expected, actual)
}

func TestAdjustBillingOnCompleteLipSyncAppliesGroupRatio(t *testing.T) {
	adaptor := &TaskAdaptor{}
	body, err := common.Marshal(map[string]any{"credits": 10})
	require.NoError(t, err)

	task := &model.Task{
		Action: constant.TaskActionLipSync,
		Data:   body,
		PrivateData: model.TaskPrivateData{
			BillingContext: &model.TaskBillingContext{
				GroupRatio: 0.5,
			},
		},
	}

	actual := adaptor.AdjustBillingOnComplete(task, nil)
	expected, _ := common.QuotaFromFloatChecked(10.0 / 10 * common.QuotaPerUnit * 0.5)
	assert.Equal(t, expected, actual)
}

func TestAdjustBillingOnCompleteLipSyncFallsBackToPrivateCredits(t *testing.T) {
	adaptor := &TaskAdaptor{}
	body, err := common.Marshal(map[string]any{"state": "success"})
	require.NoError(t, err)

	task := &model.Task{
		Action: constant.TaskActionLipSync,
		Data:   body,
		PrivateData: model.TaskPrivateData{
			UpstreamCredits: 8,
			BillingContext: &model.TaskBillingContext{
				GroupRatio: 1.0,
			},
		},
	}

	actual := adaptor.AdjustBillingOnComplete(task, nil)
	expected, _ := common.QuotaFromFloatChecked(8.0 / 10 * common.QuotaPerUnit * 1.0)
	assert.Equal(t, expected, actual)
}

func TestAdjustBillingOnCompleteLipSyncKeepsPrechargeWhenNoCredits(t *testing.T) {
	adaptor := &TaskAdaptor{}
	task := &model.Task{
		Action: constant.TaskActionLipSync,
		Data:   []byte(`{"state":"success"}`),
		PrivateData: model.TaskPrivateData{
			BillingContext: &model.TaskBillingContext{GroupRatio: 1.0},
		},
	}
	assert.Equal(t, 0, adaptor.AdjustBillingOnComplete(task, nil))
}

func TestAdjustBillingOnCompleteIgnoresNonLipSync(t *testing.T) {
	adaptor := &TaskAdaptor{}
	body, err := common.Marshal(map[string]any{"credits": 99})
	require.NoError(t, err)
	task := &model.Task{
		Action: constant.TaskActionReference2Image,
		Data:   body,
		PrivateData: model.TaskPrivateData{
			BillingContext: &model.TaskBillingContext{GroupRatio: 1.0},
		},
	}
	assert.Equal(t, 0, adaptor.AdjustBillingOnComplete(task, nil))
}
