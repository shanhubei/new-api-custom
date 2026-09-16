package baidu_vod_vidu

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	taskcommon "github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"

	"github.com/pkg/errors"
)

// ============================
// Request / Response structures
// ============================

type requestPayload struct {
	Model             string   `json:"model"`
	Images            []string `json:"images"`
	Prompt            string   `json:"prompt,omitempty"`
	Duration          int      `json:"duration,omitempty"`
	Seed              int      `json:"seed,omitempty"`
	Resolution        string   `json:"resolution,omitempty"`
	MovementAmplitude string   `json:"movement_amplitude,omitempty"`
	Bgm               bool     `json:"bgm,omitempty"`
	Payload           string   `json:"payload,omitempty"`
	CallbackUrl       string   `json:"callback_url,omitempty"`
	Moderation        *string  `json:"moderation,omitempty"`
	Audio             *bool    `json:"audio,omitempty"`
	OffPeak           *bool    `json:"off_peak,omitempty"`
}

type imageRequestPayload struct {
	Model       string   `json:"model"`
	Images      []string `json:"images,omitempty"`
	Prompt      string   `json:"prompt"`
	Seed        *int     `json:"seed,omitempty"`
	AspectRatio *string  `json:"aspect_ratio,omitempty"`
	Resolution  *string  `json:"resolution,omitempty"`
	Payload     *string  `json:"payload,omitempty"`
	CallbackUrl *string  `json:"callback_url,omitempty"`
	Moderation  *string  `json:"moderation,omitempty"`
	Audio       *bool    `json:"audio,omitempty"`
	OffPeak     *bool    `json:"off_peak,omitempty"`
}

type lipSyncRequestPayload struct {
	VideoURL    string   `json:"video_url"`
	AudioURL    string   `json:"audio_url,omitempty"`
	Text        string   `json:"text,omitempty"`
	Speed       *float64 `json:"speed,omitempty"`
	VoiceID     string   `json:"voice_id,omitempty"`
	RefPhotoURL string   `json:"ref_photo_url,omitempty"`
	Volume      *int     `json:"volume,omitempty"`
	CallbackURL string   `json:"callback_url,omitempty"`
}

const maxReferenceImages = 7

type responsePayload struct {
	TaskId            string   `json:"task_id"`
	State             string   `json:"state"`
	Model             string   `json:"model"`
	Images            []string `json:"images"`
	Prompt            string   `json:"prompt"`
	Duration          int      `json:"duration"`
	Seed              int      `json:"seed"`
	Resolution        string   `json:"resolution"`
	Bgm               bool     `json:"bgm"`
	MovementAmplitude string   `json:"movement_amplitude"`
	Payload           string   `json:"payload"`
	CreatedAt         string   `json:"created_at"`
	Credits           int      `json:"credits,omitempty"`
}

type taskResultResponse struct {
	State     string     `json:"state"`
	ErrCode   string     `json:"err_code"`
	Credits   int        `json:"credits"`
	Payload   string     `json:"payload"`
	Creations []creation `json:"creations"`
}

type creation struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	CoverURL string `json:"cover_url"`
}

// ============================
// Adaptor implementation
// ============================

type TaskAdaptor struct {
	taskcommon.BaseBilling
	ChannelType int
	baseURL     string
}

func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	a.ChannelType = info.ChannelType
	a.baseURL = info.ChannelBaseUrl
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	if isAsyncLipSyncPath(c.Request.URL.Path) {
		return a.validateLipSyncRequest(c, info)
	}
	if err := relaycommon.ValidateBasicTaskRequest(c, info, constant.TaskActionGenerate); err != nil {
		return err
	}
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return service.TaskErrorWrapper(err, "get_task_request_failed", http.StatusBadRequest)
	}
	if isAsyncImagePath(c.Request.URL.Path) {
		info.Action = constant.TaskActionReference2Image
		modelName := info.UpstreamModelName
		if modelName == "" {
			modelName = req.Model
		}
		if strings.Contains(modelName, "viduq1") && !req.HasImage() {
			return service.TaskErrorWrapperLocal(fmt.Errorf("viduq1 requires 1 to 7 images"), "missing_images", http.StatusBadRequest)
		}
		if len(req.Images) > maxReferenceImages {
			return service.TaskErrorWrapperLocal(fmt.Errorf("images must have at most %d items", maxReferenceImages), "invalid_images", http.StatusBadRequest)
		}
		return nil
	}
	action := constant.TaskActionTextGenerate
	if meatAction, ok := req.Metadata["action"]; ok {
		action, _ = meatAction.(string)
	} else if req.HasImage() {
		action = constant.TaskActionGenerate
		if info.ChannelType == constant.ChannelTypeBaiduVodVidu {
			// 百度 VOD Vidu 透传：首尾帧生视频和参考图生视频
			if len(req.Images) == 2 {
				action = constant.TaskActionFirstTailGenerate
			} else if len(req.Images) > 2 {
				action = constant.TaskActionReferenceGenerate
			}
		}
	}
	info.Action = action
	return nil
}

func (a *TaskAdaptor) validateLipSyncRequest(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	var raw map[string]any
	if err := common.UnmarshalBodyReusable(c, &raw); err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	modelName, _ := raw["model"].(string)
	if strings.TrimSpace(modelName) == "" {
		modelName = info.OriginModelName
	}
	if strings.TrimSpace(modelName) == "" {
		return service.TaskErrorWrapperLocal(fmt.Errorf("model is required"), "invalid_request", http.StatusBadRequest)
	}

	videoURL, _ := raw["video_url"].(string)
	if strings.TrimSpace(videoURL) == "" {
		return service.TaskErrorWrapperLocal(fmt.Errorf("video_url is required"), "invalid_request", http.StatusBadRequest)
	}

	audioURL, _ := raw["audio_url"].(string)
	text, _ := raw["text"].(string)
	if strings.TrimSpace(audioURL) == "" && strings.TrimSpace(text) == "" {
		return service.TaskErrorWrapperLocal(fmt.Errorf("audio_url or text is required"), "invalid_request", http.StatusBadRequest)
	}

	if v, ok := raw["volume"]; ok && v != nil {
		vol, ok := v.(float64)
		if !ok {
			return service.TaskErrorWrapperLocal(fmt.Errorf("volume must be a number"), "invalid_request", http.StatusBadRequest)
		}
		if vol < 0 || vol > 10 {
			return service.TaskErrorWrapperLocal(fmt.Errorf("volume must be between 0 and 10"), "invalid_request", http.StatusBadRequest)
		}
	}

	prompt := strings.TrimSpace(text)
	if prompt == "" {
		prompt = "[lip-sync]"
	}
	info.Action = constant.TaskActionLipSync
	c.Set("task_request", relaycommon.TaskSubmitReq{
		Prompt: prompt,
		Model:  modelName,
	})
	return nil
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	v, exists := c.Get("task_request")
	if !exists {
		return nil, fmt.Errorf("request not found in context")
	}
	req := v.(relaycommon.TaskSubmitReq)

	if info.Action == constant.TaskActionReference2Image {
		var imgBody imageRequestPayload
		if err := common.UnmarshalBodyReusable(c, &imgBody); err != nil {
			return nil, err
		}
		if strings.TrimSpace(imgBody.Model) == "" {
			imgBody.Model = info.UpstreamModelName
		}
		data, err := common.Marshal(&imgBody)
		if err != nil {
			return nil, err
		}
		return bytes.NewReader(data), nil
	}

	if info.Action == constant.TaskActionLipSync {
		var lipBody lipSyncRequestPayload
		if err := common.UnmarshalBodyReusable(c, &lipBody); err != nil {
			return nil, err
		}
		data, err := common.Marshal(&lipBody)
		if err != nil {
			return nil, err
		}
		return bytes.NewReader(data), nil
	}

	body, err := a.convertToRequestPayload(&req, info)
	if err != nil {
		return nil, err
	}

	if info.Action == constant.TaskActionReferenceGenerate {
		if strings.Contains(body.Model, "viduq2") {
			// 参考图生视频只能用 viduq2 模型, 不能带有pro或turbo后缀 https://platform.vidu.cn/docs/reference-to-video
			body.Model = "viduq2"
		}
	}

	data, err := common.Marshal(body)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(data), nil
}

func (a *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	var path string
	switch info.Action {
	case constant.TaskActionReference2Image:
		path = "/reference2image"
	case constant.TaskActionLipSync:
		path = "/lip-sync"
	case constant.TaskActionGenerate:
		path = "/img2video"
	case constant.TaskActionFirstTailGenerate:
		path = "/start-end2video"
	case constant.TaskActionReferenceGenerate:
		path = "/reference2video"
	default:
		path = "/text2video"
	}
	return fmt.Sprintf("%s/ent/v2%s", a.baseURL, path), nil
}

func (a *TaskAdaptor) BuildRequestHeader(c *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+info.ApiKey)
	return nil
}

func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (taskID string, taskData []byte, taskErr *dto.TaskError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
		return
	}

	var vResp responsePayload
	err = common.Unmarshal(responseBody, &vResp)
	if err != nil {
		taskErr = service.TaskErrorWrapper(errors.Wrap(err, fmt.Sprintf("%s", responseBody)), "unmarshal_response_failed", http.StatusInternalServerError)
		return
	}

	if vResp.State == "failed" {
		taskErr = service.TaskErrorWrapperLocal(fmt.Errorf("task failed"), "task_failed", http.StatusBadRequest)
		return
	}

	ov := dto.NewOpenAIVideo()
	ov.ID = info.PublicTaskID
	ov.TaskID = info.PublicTaskID
	ov.CreatedAt = time.Now().Unix()
	ov.Model = info.OriginModelName
	c.JSON(http.StatusOK, ov)
	return vResp.TaskId, responseBody, nil
}

func (a *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid task_id")
	}

	url := fmt.Sprintf("%s/ent/v2/tasks/%s/creations", baseUrl, taskID)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)

	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, fmt.Errorf("new proxy http client failed: %w", err)
	}
	return client.Do(req)
}

func (a *TaskAdaptor) GetModelList() []string {
	return []string{"viduq3-pro", "viduq2", "viduq1", "vidu2.0", "vidu1.5", "vidu-lip-sync"}
}

func (a *TaskAdaptor) GetChannelName() string {
	return "baidu_vod_vidu"
}

// ============================
// helpers
// ============================

func isAsyncImagePath(path string) bool {
	return strings.Contains(path, "/v1/async/images")
}

func isAsyncLipSyncPath(path string) bool {
	return strings.Contains(path, "/v1/async/lip-sync")
}

func parseLipSyncRequestPayload(data []byte) (*lipSyncRequestPayload, error) {
	var body lipSyncRequestPayload
	if err := common.Unmarshal(data, &body); err != nil {
		return nil, err
	}
	return &body, nil
}

func parseImageRequestPayload(data []byte, upstreamModel string) (*imageRequestPayload, error) {
	var body imageRequestPayload
	if err := common.Unmarshal(data, &body); err != nil {
		return nil, err
	}
	if strings.TrimSpace(body.Model) == "" {
		body.Model = upstreamModel
	}
	return &body, nil
}

func (a *TaskAdaptor) convertToRequestPayload(req *relaycommon.TaskSubmitReq, info *relaycommon.RelayInfo) (*requestPayload, error) {
	r := requestPayload{
		Model:             taskcommon.DefaultString(info.UpstreamModelName, "viduq1"),
		Images:            req.Images,
		Prompt:            req.Prompt,
		Duration:          taskcommon.DefaultInt(req.Duration, 5),
		Resolution:        taskcommon.DefaultString(req.Size, "1080p"),
		MovementAmplitude: "auto",
		Bgm:               false,
	}
	if err := taskcommon.UnmarshalMetadata(req.Metadata, &r); err != nil {
		return nil, errors.Wrap(err, "unmarshal metadata failed")
	}
	return &r, nil
}

func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	taskInfo := &relaycommon.TaskInfo{}

	var taskResp taskResultResponse
	err := common.Unmarshal(respBody, &taskResp)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal response body")
	}

	state := taskResp.State
	switch state {
	case "created", "queueing":
		taskInfo.Status = model.TaskStatusSubmitted
	case "processing":
		taskInfo.Status = model.TaskStatusInProgress
	case "success":
		taskInfo.Status = model.TaskStatusSuccess
		if len(taskResp.Creations) > 0 {
			taskInfo.Url = taskResp.Creations[0].URL
		}
	case "failed":
		taskInfo.Status = model.TaskStatusFailure
		if taskResp.ErrCode != "" {
			taskInfo.Reason = taskResp.ErrCode
		}
	default:
		return nil, fmt.Errorf("unknown task state: %s", state)
	}

	return taskInfo, nil
}

func (a *TaskAdaptor) ConvertToOpenAIVideo(originTask *model.Task) ([]byte, error) {
	var viduResp taskResultResponse
	if err := common.Unmarshal(originTask.Data, &viduResp); err != nil {
		return nil, errors.Wrap(err, "unmarshal baidu vod vidu task data failed")
	}

	openAIVideo := dto.NewOpenAIVideo()
	openAIVideo.ID = originTask.TaskID
	openAIVideo.Status = originTask.Status.ToVideoStatus()
	openAIVideo.SetProgressStr(originTask.Progress)
	openAIVideo.CreatedAt = originTask.CreatedAt
	openAIVideo.CompletedAt = originTask.UpdatedAt

	if len(viduResp.Creations) > 0 && viduResp.Creations[0].URL != "" {
		openAIVideo.SetMetadata("url", viduResp.Creations[0].URL)
	}

	if viduResp.State == "failed" && viduResp.ErrCode != "" {
		openAIVideo.Error = &dto.OpenAIVideoError{
			Message: viduResp.ErrCode,
			Code:    viduResp.ErrCode,
		}
	}

	return common.Marshal(openAIVideo)
}
