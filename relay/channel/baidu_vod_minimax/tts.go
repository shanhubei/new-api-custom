package baidu_vod_minimax

import (
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

type MiniMaxTTSRequest struct {
	Model string `json:"model"`
	Text  string `json:"text"`
	// VoiceID is required by Baidu VOD's live validator (error: voiceId must not be blank),
	// even when the documented MiniMax body uses voice_setting.voice_id.
	VoiceID           string             `json:"voiceId,omitempty"`
	Stream            bool               `json:"stream,omitempty"`
	StreamOptions     *StreamOptions     `json:"stream_options,omitempty"`
	VoiceSetting      VoiceSetting       `json:"voice_setting"`
	PronunciationDict *PronunciationDict `json:"pronunciation_dict,omitempty"`
	AudioSetting      *AudioSetting      `json:"audio_setting,omitempty"`
	TimbreWeights     []TimbreWeight     `json:"timbre_weights,omitempty"`
	LanguageBoost     string             `json:"language_boost,omitempty"`
	VoiceModify       *VoiceModify       `json:"voice_modify,omitempty"`
	SubtitleEnable    bool               `json:"subtitle_enable,omitempty"`
	OutputFormat      string             `json:"output_format,omitempty"`
	AigcWatermark     bool               `json:"aigc_watermark,omitempty"`
}

type StreamOptions struct {
	ExcludeAggregatedAudio bool `json:"exclude_aggregated_audio,omitempty"`
}

type VoiceSetting struct {
	VoiceID           string  `json:"voice_id"`
	Speed             float64 `json:"speed,omitempty"`
	Vol               float64 `json:"vol,omitempty"`
	Pitch             int     `json:"pitch,omitempty"`
	Emotion           string  `json:"emotion,omitempty"`
	TextNormalization bool    `json:"text_normalization,omitempty"`
	LatexRead         bool    `json:"latex_read,omitempty"`
}

type PronunciationDict struct {
	Tone []string `json:"tone,omitempty"`
}

type AudioSetting struct {
	SampleRate int    `json:"sample_rate,omitempty"`
	Bitrate    int    `json:"bitrate,omitempty"`
	Format     string `json:"format,omitempty"`
	Channel    int    `json:"channel,omitempty"`
	ForceCbr   bool   `json:"force_cbr,omitempty"`
}

type TimbreWeight struct {
	VoiceID string `json:"voice_id"`
	Weight  int    `json:"weight"`
}

type VoiceModify struct {
	Pitch        int    `json:"pitch,omitempty"`
	Intensity    int    `json:"intensity,omitempty"`
	Timbre       int    `json:"timbre,omitempty"`
	SoundEffects string `json:"sound_effects,omitempty"`
}

type MiniMaxTTSResponse struct {
	Data      MiniMaxTTSData   `json:"data"`
	ExtraInfo MiniMaxExtraInfo `json:"extra_info"`
	TraceID   string           `json:"trace_id"`
	BaseResp  MiniMaxBaseResp  `json:"base_resp"`
}

type MiniMaxTTSData struct {
	Audio  string `json:"audio"`
	Status int    `json:"status"`
}

type MiniMaxExtraInfo struct {
	UsageCharacters int64 `json:"usage_characters"`
}

type MiniMaxBaseResp struct {
	StatusCode int64  `json:"status_code"`
	StatusMsg  string `json:"status_msg"`
}

// camelCase alternate used by some Baidu Java gateways.
type minimaxTTSResponseCamel struct {
	Data      MiniMaxTTSData `json:"data"`
	ExtraInfo struct {
		UsageCharacters int64 `json:"usageCharacters"`
	} `json:"extraInfo"`
	TraceID  string `json:"traceId"`
	BaseResp struct {
		StatusCode int64  `json:"statusCode"`
		StatusMsg  string `json:"statusMsg"`
	} `json:"baseResp"`
}

type parsedTTSResponse struct {
	Audio           string
	UsageCharacters int64
	StatusCode      int64
	StatusMsg       string
}

func parseTTSResponse(body []byte) (parsedTTSResponse, error) {
	var out parsedTTSResponse

	var snake MiniMaxTTSResponse
	if err := common.Unmarshal(body, &snake); err != nil {
		return out, err
	}
	out.Audio = strings.TrimSpace(snake.Data.Audio)
	out.UsageCharacters = snake.ExtraInfo.UsageCharacters
	out.StatusCode = snake.BaseResp.StatusCode
	out.StatusMsg = snake.BaseResp.StatusMsg

	if out.Audio == "" || (out.StatusCode == 0 && out.StatusMsg == "" && out.UsageCharacters == 0) {
		var camel minimaxTTSResponseCamel
		if err := common.Unmarshal(body, &camel); err == nil {
			if out.Audio == "" {
				out.Audio = strings.TrimSpace(camel.Data.Audio)
			}
			if out.UsageCharacters == 0 {
				out.UsageCharacters = camel.ExtraInfo.UsageCharacters
			}
			if out.StatusCode == 0 && camel.BaseResp.StatusCode != 0 {
				out.StatusCode = camel.BaseResp.StatusCode
			}
			if out.StatusMsg == "" {
				out.StatusMsg = camel.BaseResp.StatusMsg
			}
		}
	}

	if out.Audio == "" {
		var raw map[string]any
		if err := common.Unmarshal(body, &raw); err == nil {
			if audio, ok := raw["audio"].(string); ok {
				out.Audio = strings.TrimSpace(audio)
			}
			// Baidu VOD live response for output_format=url may be {"url":"https://..."}.
			if out.Audio == "" {
				if u, ok := raw["url"].(string); ok {
					out.Audio = strings.TrimSpace(u)
				}
			}
			if out.Audio == "" {
				if data, ok := raw["data"].(map[string]any); ok {
					if audio, ok := data["audio"].(string); ok {
						out.Audio = strings.TrimSpace(audio)
					}
					if out.Audio == "" {
						if u, ok := data["url"].(string); ok {
							out.Audio = strings.TrimSpace(u)
						}
					}
				}
			}
			// Baidu BCE-style error payload.
			if code, ok := raw["code"].(string); ok && code != "" && !strings.EqualFold(code, "ok") {
				msg, _ := raw["message"].(string)
				if msg == "" {
					msg = code
				}
				out.StatusCode = 400
				out.StatusMsg = msg
			}
		}
	}

	return out, nil
}

func truncateForError(body []byte, max int) string {
	s := strings.TrimSpace(string(body))
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func ConvertOpenAIAudioToTTSRequest(info *relaycommon.RelayInfo, request dto.AudioRequest) ([]byte, string, error) {
	voiceID := strings.TrimSpace(request.Voice)
	text := strings.TrimSpace(request.Input)
	speed := lo.FromPtrOr(request.Speed, 0.0)
	audioFormat := request.ResponseFormat

	modelName := info.OriginModelName
	if info.ChannelMeta != nil && info.ChannelMeta.UpstreamModelName != "" {
		modelName = info.ChannelMeta.UpstreamModelName
	}

	ttsRequest := MiniMaxTTSRequest{
		Model:   modelName,
		Text:    text,
		VoiceID: voiceID,
		VoiceSetting: VoiceSetting{
			VoiceID: voiceID,
			Speed:   speed,
		},
		AudioSetting: &AudioSetting{
			Format: audioFormat,
		},
		OutputFormat: audioFormat,
	}

	if len(request.Metadata) > 0 {
		if err := common.Unmarshal(request.Metadata, &ttsRequest); err != nil {
			return nil, "", fmt.Errorf("error unmarshalling metadata to TTS request: %w", err)
		}
	}

	// metadata 可能冲掉 OpenAI 映射；回填 text / MiniMax voice_id / 百度顶层 voiceId。
	if text != "" {
		ttsRequest.Text = text
	}
	if voiceID != "" {
		ttsRequest.VoiceID = voiceID
		ttsRequest.VoiceSetting.VoiceID = voiceID
	} else if ttsRequest.VoiceSetting.VoiceID != "" {
		ttsRequest.VoiceID = ttsRequest.VoiceSetting.VoiceID
	} else if ttsRequest.VoiceID != "" {
		ttsRequest.VoiceSetting.VoiceID = ttsRequest.VoiceID
	}

	if strings.TrimSpace(ttsRequest.Text) == "" {
		return nil, "", fmt.Errorf("input is required")
	}
	if strings.TrimSpace(ttsRequest.VoiceID) == "" && strings.TrimSpace(ttsRequest.VoiceSetting.VoiceID) == "" {
		return nil, "", fmt.Errorf("voice is required")
	}
	if ttsRequest.VoiceID == "" {
		ttsRequest.VoiceID = ttsRequest.VoiceSetting.VoiceID
	}
	if ttsRequest.VoiceSetting.VoiceID == "" {
		ttsRequest.VoiceSetting.VoiceID = ttsRequest.VoiceID
	}

	outFmt := audioFormat
	if outFmt != "hex" {
		outFmt = "url"
	}
	ttsRequest.OutputFormat = outFmt

	jsonData, err := common.Marshal(ttsRequest)
	if err != nil {
		return nil, "", fmt.Errorf("error marshalling TTS request: %w", err)
	}
	return jsonData, outFmt, nil
}

func HandleTTSResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (usage any, err *types.NewAPIError) {
	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, types.NewErrorWithStatusCode(
			fmt.Errorf("failed to read baidu vod minimax response: %w", readErr),
			types.ErrorCodeReadResponseBodyFailed,
			http.StatusInternalServerError,
		)
	}
	defer resp.Body.Close()

	ttsResp, parseErr := parseTTSResponse(body)
	if parseErr != nil {
		return nil, types.NewErrorWithStatusCode(
			fmt.Errorf("failed to unmarshal baidu vod minimax TTS response: %w; body=%s", parseErr, truncateForError(body, 512)),
			types.ErrorCodeBadResponseBody,
			http.StatusInternalServerError,
		)
	}

	if ttsResp.StatusCode != 0 {
		return nil, types.NewErrorWithStatusCode(
			fmt.Errorf("baidu vod minimax TTS error: %d - %s", ttsResp.StatusCode, ttsResp.StatusMsg),
			types.ErrorCodeBadResponse,
			http.StatusBadRequest,
		)
	}

	if ttsResp.Audio == "" {
		return nil, types.NewErrorWithStatusCode(
			fmt.Errorf("no audio data in baidu vod minimax TTS response; body=%s", truncateForError(body, 512)),
			types.ErrorCodeBadResponse,
			http.StatusBadRequest,
		)
	}

	if strings.HasPrefix(ttsResp.Audio, "http") {
		// URL 模式直接 JSON 回传，便于客户端取地址；hex 模式仍返回音频二进制。
		c.JSON(http.StatusOK, gin.H{"url": ttsResp.Audio})
	} else {
		audioData, decodeErr := hex.DecodeString(ttsResp.Audio)
		if decodeErr != nil {
			return nil, types.NewErrorWithStatusCode(
				fmt.Errorf("failed to decode hex audio data: %w", decodeErr),
				types.ErrorCodeBadResponse,
				http.StatusInternalServerError,
			)
		}
		c.Data(http.StatusOK, "audio/mpeg", audioData)
	}

	usage = &dto.Usage{
		PromptTokens:     info.GetEstimatePromptTokens(),
		CompletionTokens: 0,
		TotalTokens:      int(ttsResp.UsageCharacters),
	}

	return usage, nil
}
