# Review Package Task 3
BASE: 9cba93a3d7ee19bcf56ee19341895872313f8c63
HEAD: aad9917cfe9feceb54fffacec2fcc8f3ead84086

## Commits
aad9917c feat(baidu-vod-minimax): implement TTS convert and response

## Stat
 relay/channel/baidu_vod_minimax/adaptor.go  |  15 ++-
 relay/channel/baidu_vod_minimax/tts.go      | 187 ++++++++++++++++++++++++++++
 relay/channel/baidu_vod_minimax/tts_test.go |  66 ++++++++++
 3 files changed, 267 insertions(+), 1 deletion(-)

## Diff
```
diff --git a/relay/channel/baidu_vod_minimax/adaptor.go b/relay/channel/baidu_vod_minimax/adaptor.go
index 70ef846f..bfc49f91 100644
--- a/relay/channel/baidu_vod_minimax/adaptor.go
+++ b/relay/channel/baidu_vod_minimax/adaptor.go
@@ -1,22 +1,24 @@
 package baidu_vod_minimax
 
 import (
+	"bytes"
 	"errors"
 	"fmt"
 	"io"
 	"net/http"
 	"strings"
 	"time"
 
 	channelconstant "github.com/QuantumNous/new-api/constant"
 	"github.com/QuantumNous/new-api/dto"
 	"github.com/QuantumNous/new-api/relay/channel"
 	relaycommon "github.com/QuantumNous/new-api/relay/common"
+	"github.com/QuantumNous/new-api/relay/constant"
 	"github.com/QuantumNous/new-api/types"
 
 	"github.com/gin-gonic/gin"
 )
 
 type Adaptor struct {
 }
 
@@ -24,17 +26,25 @@ func (a *Adaptor) ConvertGeminiRequest(*gin.Context, *relaycommon.RelayInfo, *dt
 	return nil, errors.New("not implemented")
 }
 
 func (a *Adaptor) ConvertClaudeRequest(*gin.Context, *relaycommon.RelayInfo, *dto.ClaudeRequest) (any, error) {
 	return nil, errors.New("not implemented")
 }
 
 func (a *Adaptor) ConvertAudioRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.AudioRequest) (io.Reader, error) {
-	return nil, errors.New("not implemented")
+	if info.RelayMode != constant.RelayModeAudioSpeech {
+		return nil, errors.New("unsupported audio relay mode")
+	}
+	raw, outFmt, err := ConvertOpenAIAudioToTTSRequest(info, request)
+	if err != nil {
+		return nil, err
+	}
+	c.Set("response_format", outFmt)
+	return bytes.NewReader(raw), nil
 }
 
 func (a *Adaptor) ConvertImageRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.ImageRequest) (any, error) {
 	return nil, errors.New("not implemented")
 }
 
 func (a *Adaptor) Init(info *relaycommon.RelayInfo) {
 }
@@ -78,16 +88,19 @@ func (a *Adaptor) ConvertOpenAIResponsesRequest(c *gin.Context, info *relaycommo
 	return nil, errors.New("not implemented")
 }
 
 func (a *Adaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (any, error) {
 	return channel.DoApiRequest(a, c, info, requestBody)
 }
 
 func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (usage any, err *types.NewAPIError) {
+	if info.RelayMode == constant.RelayModeAudioSpeech {
+		return HandleTTSResponse(c, resp, info)
+	}
 	return nil, types.NewError(errors.New("not implemented"), types.ErrorCodeInvalidRequest)
 }
 
 func (a *Adaptor) GetModelList() []string {
 	return ModelList
 }
 
 func (a *Adaptor) GetChannelName() string {
diff --git a/relay/channel/baidu_vod_minimax/tts.go b/relay/channel/baidu_vod_minimax/tts.go
new file mode 100644
index 00000000..e8cd4586
--- /dev/null
+++ b/relay/channel/baidu_vod_minimax/tts.go
@@ -0,0 +1,187 @@
+package baidu_vod_minimax
+
+import (
+	"encoding/hex"
+	"fmt"
+	"io"
+	"net/http"
+	"strings"
+
+	"github.com/QuantumNous/new-api/common"
+	"github.com/QuantumNous/new-api/dto"
+	relaycommon "github.com/QuantumNous/new-api/relay/common"
+	"github.com/QuantumNous/new-api/types"
+	"github.com/gin-gonic/gin"
+	"github.com/samber/lo"
+)
+
+type MiniMaxTTSRequest struct {
+	Model             string             `json:"model"`
+	Text              string             `json:"text"`
+	Stream            bool               `json:"stream,omitempty"`
+	StreamOptions     *StreamOptions     `json:"stream_options,omitempty"`
+	VoiceSetting      VoiceSetting       `json:"voice_setting"`
+	PronunciationDict *PronunciationDict `json:"pronunciation_dict,omitempty"`
+	AudioSetting      *AudioSetting      `json:"audio_setting,omitempty"`
+	TimbreWeights     []TimbreWeight     `json:"timbre_weights,omitempty"`
+	LanguageBoost     string             `json:"language_boost,omitempty"`
+	VoiceModify       *VoiceModify       `json:"voice_modify,omitempty"`
+	SubtitleEnable    bool               `json:"subtitle_enable,omitempty"`
+	OutputFormat      string             `json:"output_format,omitempty"`
+	AigcWatermark     bool               `json:"aigc_watermark,omitempty"`
+}
+
+type StreamOptions struct {
+	ExcludeAggregatedAudio bool `json:"exclude_aggregated_audio,omitempty"`
+}
+
+type VoiceSetting struct {
+	VoiceID           string  `json:"voice_id"`
+	Speed             float64 `json:"speed,omitempty"`
+	Vol               float64 `json:"vol,omitempty"`
+	Pitch             int     `json:"pitch,omitempty"`
+	Emotion           string  `json:"emotion,omitempty"`
+	TextNormalization bool    `json:"text_normalization,omitempty"`
+	LatexRead         bool    `json:"latex_read,omitempty"`
+}
+
+type PronunciationDict struct {
+	Tone []string `json:"tone,omitempty"`
+}
+
+type AudioSetting struct {
+	SampleRate int    `json:"sample_rate,omitempty"`
+	Bitrate    int    `json:"bitrate,omitempty"`
+	Format     string `json:"format,omitempty"`
+	Channel    int    `json:"channel,omitempty"`
+	ForceCbr   bool   `json:"force_cbr,omitempty"`
+}
+
+type TimbreWeight struct {
+	VoiceID string `json:"voice_id"`
+	Weight  int    `json:"weight"`
+}
+
+type VoiceModify struct {
+	Pitch        int    `json:"pitch,omitempty"`
+	Intensity    int    `json:"intensity,omitempty"`
+	Timbre       int    `json:"timbre,omitempty"`
+	SoundEffects string `json:"sound_effects,omitempty"`
+}
+
+type MiniMaxTTSResponse struct {
+	Data      MiniMaxTTSData   `json:"data"`
+	ExtraInfo MiniMaxExtraInfo `json:"extra_info"`
+	TraceID   string           `json:"trace_id"`
+	BaseResp  MiniMaxBaseResp  `json:"base_resp"`
+}
+
+type MiniMaxTTSData struct {
+	Audio  string `json:"audio"`
+	Status int    `json:"status"`
+}
+
+type MiniMaxExtraInfo struct {
+	UsageCharacters int64 `json:"usage_characters"`
+}
+
+type MiniMaxBaseResp struct {
+	StatusCode int64  `json:"status_code"`
+	StatusMsg  string `json:"status_msg"`
+}
+
+func ConvertOpenAIAudioToTTSRequest(info *relaycommon.RelayInfo, request dto.AudioRequest) ([]byte, string, error) {
+	voiceID := request.Voice
+	speed := lo.FromPtrOr(request.Speed, 0.0)
+	audioFormat := request.ResponseFormat
+
+	ttsRequest := MiniMaxTTSRequest{
+		Model: info.OriginModelName,
+		Text:  request.Input,
+		VoiceSetting: VoiceSetting{
+			VoiceID: voiceID,
+			Speed:   speed,
+		},
+		AudioSetting: &AudioSetting{
+			Format: audioFormat,
+		},
+		OutputFormat: audioFormat,
+	}
+
+	if len(request.Metadata) > 0 {
+		if err := common.Unmarshal(request.Metadata, &ttsRequest); err != nil {
+			return nil, "", fmt.Errorf("error unmarshalling metadata to TTS request: %w", err)
+		}
+	}
+
+	outFmt := audioFormat
+	if outFmt != "hex" {
+		outFmt = "url"
+	}
+	ttsRequest.OutputFormat = outFmt
+
+	jsonData, err := common.Marshal(ttsRequest)
+	if err != nil {
+		return nil, "", fmt.Errorf("error marshalling TTS request: %w", err)
+	}
+	return jsonData, outFmt, nil
+}
+
+func HandleTTSResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (usage any, err *types.NewAPIError) {
+	body, readErr := io.ReadAll(resp.Body)
+	if readErr != nil {
+		return nil, types.NewErrorWithStatusCode(
+			fmt.Errorf("failed to read baidu vod minimax response: %w", readErr),
+			types.ErrorCodeReadResponseBodyFailed,
+			http.StatusInternalServerError,
+		)
+	}
+	defer resp.Body.Close()
+
+	var ttsResp MiniMaxTTSResponse
+	if unmarshalErr := common.Unmarshal(body, &ttsResp); unmarshalErr != nil {
+		return nil, types.NewErrorWithStatusCode(
+			fmt.Errorf("failed to unmarshal baidu vod minimax TTS response: %w", unmarshalErr),
+			types.ErrorCodeBadResponseBody,
+			http.StatusInternalServerError,
+		)
+	}
+
+	if ttsResp.BaseResp.StatusCode != 0 {
+		return nil, types.NewErrorWithStatusCode(
+			fmt.Errorf("baidu vod minimax TTS error: %d - %s", ttsResp.BaseResp.StatusCode, ttsResp.BaseResp.StatusMsg),
+			types.ErrorCodeBadResponse,
+			http.StatusBadRequest,
+		)
+	}
+
+	if ttsResp.Data.Audio == "" {
+		return nil, types.NewErrorWithStatusCode(
+			fmt.Errorf("no audio data in baidu vod minimax TTS response"),
+			types.ErrorCodeBadResponse,
+			http.StatusBadRequest,
+		)
+	}
+
+	if strings.HasPrefix(ttsResp.Data.Audio, "http") {
+		c.Redirect(http.StatusFound, ttsResp.Data.Audio)
+	} else {
+		audioData, decodeErr := hex.DecodeString(ttsResp.Data.Audio)
+		if decodeErr != nil {
+			return nil, types.NewErrorWithStatusCode(
+				fmt.Errorf("failed to decode hex audio data: %w", decodeErr),
+				types.ErrorCodeBadResponse,
+				http.StatusInternalServerError,
+			)
+		}
+		c.Data(http.StatusOK, "audio/mpeg", audioData)
+	}
+
+	usage = &dto.Usage{
+		PromptTokens:     info.GetEstimatePromptTokens(),
+		CompletionTokens: 0,
+		TotalTokens:      int(ttsResp.ExtraInfo.UsageCharacters),
+	}
+
+	return usage, nil
+}
diff --git a/relay/channel/baidu_vod_minimax/tts_test.go b/relay/channel/baidu_vod_minimax/tts_test.go
new file mode 100644
index 00000000..c3b87247
--- /dev/null
+++ b/relay/channel/baidu_vod_minimax/tts_test.go
@@ -0,0 +1,66 @@
+package baidu_vod_minimax
+
+import (
+	"io"
+	"net/http"
+	"net/http/httptest"
+	"strings"
+	"testing"
+
+	"github.com/QuantumNous/new-api/common"
+	"github.com/QuantumNous/new-api/dto"
+	relaycommon "github.com/QuantumNous/new-api/relay/common"
+	"github.com/gin-gonic/gin"
+	"github.com/stretchr/testify/assert"
+	"github.com/stretchr/testify/require"
+)
+
+func TestConvertOpenAIAudioToTTSRequestMapsFields(t *testing.T) {
+	speed := 1.2
+	req := dto.AudioRequest{
+		Model:          "speech-2.8-hd",
+		Input:          "浣犲ソ",
+		Voice:          "Boyan_new_hd",
+		Speed:          &speed,
+		ResponseFormat: "mp3",
+	}
+	info := &relaycommon.RelayInfo{OriginModelName: "speech-2.8-hd"}
+	raw, outFmt, err := ConvertOpenAIAudioToTTSRequest(info, req)
+	require.NoError(t, err)
+	assert.Equal(t, "url", outFmt)
+
+	var body map[string]any
+	require.NoError(t, common.Unmarshal(raw, &body))
+	assert.Equal(t, "speech-2.8-hd", body["model"])
+	assert.Equal(t, "浣犲ソ", body["text"])
+	assert.Equal(t, "url", body["output_format"])
+	vs := body["voice_setting"].(map[string]any)
+	assert.Equal(t, "Boyan_new_hd", vs["voice_id"])
+}
+
+func TestHandleTTSResponseURL(t *testing.T) {
+	gin.SetMode(gin.TestMode)
+	w := httptest.NewRecorder()
+	c, _ := gin.CreateTestContext(w)
+	c.Request = httptest.NewRequest(http.MethodGet, "/v1/audio/speech", nil)
+	body := `{"data":{"audio":"https://example.com/a.mp3","status":1},"extra_info":{"usage_characters":20},"base_resp":{"status_code":0,"status_msg":"OK"}}`
+	resp := &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}
+	info := &relaycommon.RelayInfo{}
+	usage, err := HandleTTSResponse(c, resp, info)
+	require.Nil(t, err)
+	u := usage.(*dto.Usage)
+	assert.Equal(t, 20, u.TotalTokens)
+	assert.Equal(t, http.StatusFound, w.Code)
+}
+
+func TestHandleTTSResponseErrorStatus(t *testing.T) {
+	gin.SetMode(gin.TestMode)
+	w := httptest.NewRecorder()
+	c, _ := gin.CreateTestContext(w)
+	body := `{"data":{"audio":"","status":0},"extra_info":{"usage_characters":0},"base_resp":{"status_code":1001,"status_msg":"invalid request"}}`
+	resp := &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}
+	info := &relaycommon.RelayInfo{}
+	usage, err := HandleTTSResponse(c, resp, info)
+	require.NotNil(t, err)
+	assert.Nil(t, usage)
+}

```
