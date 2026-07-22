package ali

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertToAliRequestVideoEditKeepsDurationZero(t *testing.T) {
	adaptor := &TaskAdaptor{}
	req := relaycommon.TaskSubmitReq{
		Model:  "wan2.7-videoedit",
		Prompt: "黏土风格",
		Size:   "720P",
		Metadata: map[string]interface{}{
			"input": map[string]interface{}{
				"media": []interface{}{
					map[string]interface{}{
						"type": "video",
						"url":  "https://example.com/source.mp4",
					},
				},
			},
			"parameters": map[string]interface{}{
				"resolution": "720P",
				"duration":   0,
			},
		},
	}

	aliReq, err := adaptor.convertToAliRequest(testRelayInfo(), req)

	require.NoError(t, err)
	require.Equal(t, 0, aliReq.Parameters.Duration)
	require.Equal(t, []AliVideoMedia{
		{Type: "video", URL: "https://example.com/source.mp4"},
	}, aliReq.Input.Media)

	body, err := common.Marshal(aliReq)
	require.NoError(t, err)
	require.NotContains(t, string(body), `"duration"`)
}

func TestConvertToAliRequestVideoEditRequiresVideoMedia(t *testing.T) {
	adaptor := &TaskAdaptor{}
	req := relaycommon.TaskSubmitReq{
		Model:  "wan2.7-videoedit",
		Prompt: "edit",
		Metadata: map[string]interface{}{
			"input": map[string]interface{}{
				"media": []interface{}{
					map[string]interface{}{
						"type": "reference_image",
						"url":  "https://example.com/ref.png",
					},
				},
			},
		},
	}

	_, err := adaptor.convertToAliRequest(testRelayInfo(), req)
	require.Error(t, err)
	require.Contains(t, err.Error(), "exactly one media with type=video")
}

func TestConvertToAliRequestR2VAcceptsReferenceMedia(t *testing.T) {
	adaptor := &TaskAdaptor{}
	req := relaycommon.TaskSubmitReq{
		Model:    "wan2.7-r2v",
		Prompt:   "character1 walks",
		Duration: 5,
		Size:     "1080P",
		Metadata: map[string]interface{}{
			"input": map[string]interface{}{
				"media": []interface{}{
					map[string]interface{}{
						"type":            "reference_image",
						"url":             "https://example.com/girl.jpg",
						"reference_voice": "https://example.com/girl.mp3",
					},
					map[string]interface{}{
						"type": "reference_video",
						"url":  "https://example.com/boy.mp4",
					},
				},
			},
			"parameters": map[string]interface{}{
				"ratio": "16:9",
			},
		},
	}

	aliReq, err := adaptor.convertToAliRequest(testRelayInfo(), req)

	require.NoError(t, err)
	require.Equal(t, 5, aliReq.Parameters.Duration)
	require.Equal(t, "16:9", aliReq.Parameters.Ratio)
	require.Equal(t, []AliVideoMedia{
		{Type: "reference_image", URL: "https://example.com/girl.jpg", ReferenceVoice: "https://example.com/girl.mp3"},
		{Type: "reference_video", URL: "https://example.com/boy.mp4"},
	}, aliReq.Input.Media)
}

func TestConvertToAliRequestR2VRequiresReferenceMedia(t *testing.T) {
	adaptor := &TaskAdaptor{}
	req := relaycommon.TaskSubmitReq{
		Model:  "wan2.7-r2v",
		Prompt: "missing refs",
	}

	_, err := adaptor.convertToAliRequest(testRelayInfo(), req)
	require.Error(t, err)
	require.Contains(t, err.Error(), "reference_image or reference_video")
}

func TestEstimateBillableSecondsVideoEditAndR2V(t *testing.T) {
	assert.Equal(t, 20, estimateBillableSeconds(&AliVideoRequest{
		Model:      "wan2.7-videoedit",
		Parameters: &AliVideoParameters{Duration: 0},
	}))
	assert.Equal(t, 10, estimateBillableSeconds(&AliVideoRequest{
		Model:      "wan2.7-videoedit",
		Parameters: &AliVideoParameters{Duration: 5},
	}))
	assert.Equal(t, 10, estimateBillableSeconds(&AliVideoRequest{
		Model:      "wan2.7-r2v",
		Parameters: &AliVideoParameters{Duration: 5},
	}))
	assert.Equal(t, 5, estimateBillableSeconds(&AliVideoRequest{
		Model:      "wan2.7-t2v",
		Parameters: &AliVideoParameters{Duration: 5},
	}))
}

func TestAdjustBillingOnCompleteUsesUsageDuration(t *testing.T) {
	adaptor := &TaskAdaptor{}
	body, err := common.Marshal(AliVideoResponse{
		Usage: &AliUsage{
			Duration:            10.04,
			InputVideoDuration:  5.02,
			OutputVideoDuration: 5.02,
		},
	})
	require.NoError(t, err)

	task := &model.Task{
		Properties: model.Properties{OriginModelName: "wan2.7-videoedit"},
		Data:       body,
		PrivateData: model.TaskPrivateData{
			BillingContext: &model.TaskBillingContext{
				ModelPrice:      0.1,
				GroupRatio:      1,
				OriginModelName: "wan2.7-videoedit",
				OtherRatios: map[string]float64{
					"seconds": 20,
				},
			},
		},
	}

	actual := adaptor.AdjustBillingOnComplete(task, nil)
	expected, _ := common.QuotaFromFloatChecked(0.1 * common.QuotaPerUnit * 1 * 11)
	require.Equal(t, expected, actual)
}

func TestParseAliUsageBillableSecondsFallsBackToInputPlusOutput(t *testing.T) {
	body, err := common.Marshal(AliVideoResponse{
		Usage: &AliUsage{
			InputVideoDuration:  3.2,
			OutputVideoDuration: 4.8,
		},
	})
	require.NoError(t, err)
	require.Equal(t, 8, parseAliUsageBillableSeconds(body))
}
