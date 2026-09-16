# Final Branch Review Package
MERGE_BASE: 6548aa478715a3c7952b7fdf7f1351d9341a772a
HEAD: 50eb43f661ef3eac4762b418f8b2461b03090a27

## Commits
50eb43f6 feat(baidu-vod-minimax): expose channel type in admin UI
21f97bd3 feat(baidu-vod-minimax): register models and header smoke test
aad9917c feat(baidu-vod-minimax): implement TTS convert and response
9cba93a3 feat(baidu-vod-minimax): wire channel type 60 and adaptor stub
594502be fix(baidu-vod-minimax): golden-vector test and trim host for BCE sign
ff76b2d4 feat(baidu-vod-minimax): add BCE bce-auth-v1 signer

## Stat
 common/api_type.go                                 |   2 +
 constant/api_type.go                               |   1 +
 constant/channel.go                                | 229 +++++++++++----------
 controller/model.go                                |   9 +
 model/pricing_default.go                           |   3 +-
 relay/channel/baidu_vod_minimax/adaptor.go         | 108 ++++++++++
 relay/channel/baidu_vod_minimax/adaptor_test.go    |  49 +++++
 relay/channel/baidu_vod_minimax/constants.go       |  14 ++
 relay/channel/baidu_vod_minimax/sign.go            | 135 ++++++++++++
 relay/channel/baidu_vod_minimax/sign_test.go       | 103 +++++++++
 relay/channel/baidu_vod_minimax/tts.go             | 187 +++++++++++++++++
 relay/channel/baidu_vod_minimax/tts_test.go        |  66 ++++++
 relay/relay_adaptor.go                             |   5 +-
 web/classic/src/constants/channel.constants.js     |   5 +
 web/default/src/features/channels/constants.ts     |   3 +-
 .../src/features/channels/lib/channel-utils.ts     |   1 +
 16 files changed, 804 insertions(+), 116 deletions(-)

## Diff
```
diff --git a/common/api_type.go b/common/api_type.go
index c198ffc0..549e2fff 100644
--- a/common/api_type.go
+++ b/common/api_type.go
@@ -75,10 +75,12 @@ func ChannelType2APIType(channelType int) (int, bool) {
 		apiType = constant.APITypeReplicate
 	case constant.ChannelTypeCodex:
 		apiType = constant.APITypeCodex
 	case constant.ChannelTypeAdvancedCustom:
 		apiType = constant.APITypeAdvancedCustom
+	case constant.ChannelTypeBaiduVodMinimax:
+		apiType = constant.APITypeBaiduVodMinimax
 	}
 	if apiType == -1 {
 		return constant.APITypeOpenAI, false
 	}
 	return apiType, true
diff --git a/constant/api_type.go b/constant/api_type.go
index f3657a11..90116ca7 100644
--- a/constant/api_type.go
+++ b/constant/api_type.go
@@ -35,7 +35,8 @@ const (
 	APITypeSubmodel
 	APITypeMiniMax
 	APITypeReplicate
 	APITypeCodex
 	APITypeAdvancedCustom
+	APITypeBaiduVodMinimax
 	APITypeDummy // this one is only for count, do not add any channel after this
 )
diff --git a/constant/channel.go b/constant/channel.go
index b30c9a73..ce7ab9bd 100644
--- a/constant/channel.go
+++ b/constant/channel.go
@@ -1,65 +1,66 @@
 package constant
 
 const (
-	ChannelTypeUnknown        = 0
-	ChannelTypeOpenAI         = 1
-	ChannelTypeMidjourney     = 2
-	ChannelTypeAzure          = 3
-	ChannelTypeOllama         = 4
-	ChannelTypeMidjourneyPlus = 5
-	ChannelTypeOpenAIMax      = 6
-	ChannelTypeOhMyGPT        = 7
-	ChannelTypeCustom         = 8
-	ChannelTypeAILS           = 9
-	ChannelTypeAIProxy        = 10
-	ChannelTypePaLM           = 11
-	ChannelTypeAPI2GPT        = 12
-	ChannelTypeAIGC2D         = 13
-	ChannelTypeAnthropic      = 14
-	ChannelTypeBaidu          = 15
-	ChannelTypeZhipu          = 16
-	ChannelTypeAli            = 17
-	ChannelTypeXunfei         = 18
-	ChannelType360            = 19
-	ChannelTypeOpenRouter     = 20
-	ChannelTypeAIProxyLibrary = 21
-	ChannelTypeFastGPT        = 22
-	ChannelTypeTencent        = 23
-	ChannelTypeGemini         = 24
-	ChannelTypeMoonshot       = 25
-	ChannelTypeZhipu_v4       = 26
-	ChannelTypePerplexity     = 27
-	ChannelTypeLingYiWanWu    = 31
-	ChannelTypeAws            = 33
-	ChannelTypeCohere         = 34
-	ChannelTypeMiniMax        = 35
-	ChannelTypeSunoAPI        = 36
-	ChannelTypeDify           = 37
-	ChannelTypeJina           = 38
-	ChannelCloudflare         = 39
-	ChannelTypeSiliconFlow    = 40
-	ChannelTypeVertexAi       = 41
-	ChannelTypeMistral        = 42
-	ChannelTypeDeepSeek       = 43
-	ChannelTypeMokaAI         = 44
-	ChannelTypeVolcEngine     = 45
-	ChannelTypeBaiduV2        = 46
-	ChannelTypeXinference     = 47
-	ChannelTypeXai            = 48
-	ChannelTypeCoze           = 49
-	ChannelTypeKling          = 50
-	ChannelTypeJimeng         = 51
-	ChannelTypeVidu           = 52
-	ChannelTypeSubmodel       = 53
-	ChannelTypeDoubaoVideo    = 54
-	ChannelTypeSora           = 55
-	ChannelTypeReplicate      = 56
-	ChannelTypeCodex          = 57
-	ChannelTypeAdvancedCustom = 58
-	ChannelTypeBaiduVodVidu   = 59
-	ChannelTypeDummy          // this one is only for count, do not add any channel after this
+	ChannelTypeUnknown         = 0
+	ChannelTypeOpenAI          = 1
+	ChannelTypeMidjourney      = 2
+	ChannelTypeAzure           = 3
+	ChannelTypeOllama          = 4
+	ChannelTypeMidjourneyPlus  = 5
+	ChannelTypeOpenAIMax       = 6
+	ChannelTypeOhMyGPT         = 7
+	ChannelTypeCustom          = 8
+	ChannelTypeAILS            = 9
+	ChannelTypeAIProxy         = 10
+	ChannelTypePaLM            = 11
+	ChannelTypeAPI2GPT         = 12
+	ChannelTypeAIGC2D          = 13
+	ChannelTypeAnthropic       = 14
+	ChannelTypeBaidu           = 15
+	ChannelTypeZhipu           = 16
+	ChannelTypeAli             = 17
+	ChannelTypeXunfei          = 18
+	ChannelType360             = 19
+	ChannelTypeOpenRouter      = 20
+	ChannelTypeAIProxyLibrary  = 21
+	ChannelTypeFastGPT         = 22
+	ChannelTypeTencent         = 23
+	ChannelTypeGemini          = 24
+	ChannelTypeMoonshot        = 25
+	ChannelTypeZhipu_v4        = 26
+	ChannelTypePerplexity      = 27
+	ChannelTypeLingYiWanWu     = 31
+	ChannelTypeAws             = 33
+	ChannelTypeCohere          = 34
+	ChannelTypeMiniMax         = 35
+	ChannelTypeSunoAPI         = 36
+	ChannelTypeDify            = 37
+	ChannelTypeJina            = 38
+	ChannelCloudflare          = 39
+	ChannelTypeSiliconFlow     = 40
+	ChannelTypeVertexAi        = 41
+	ChannelTypeMistral         = 42
+	ChannelTypeDeepSeek        = 43
+	ChannelTypeMokaAI          = 44
+	ChannelTypeVolcEngine      = 45
+	ChannelTypeBaiduV2         = 46
+	ChannelTypeXinference      = 47
+	ChannelTypeXai             = 48
+	ChannelTypeCoze            = 49
+	ChannelTypeKling           = 50
+	ChannelTypeJimeng          = 51
+	ChannelTypeVidu            = 52
+	ChannelTypeSubmodel        = 53
+	ChannelTypeDoubaoVideo     = 54
+	ChannelTypeSora            = 55
+	ChannelTypeReplicate       = 56
+	ChannelTypeCodex           = 57
+	ChannelTypeAdvancedCustom  = 58
+	ChannelTypeBaiduVodVidu    = 59
+	ChannelTypeBaiduVodMinimax = 60
+	ChannelTypeDummy           // this one is only for count, do not add any channel after this
 
 )
 
 var ChannelBaseURLs = []string{
 	"",                                    // 0
@@ -120,69 +121,71 @@ var ChannelBaseURLs = []string{
 	"https://api.openai.com",                    //55
 	"https://api.replicate.com",                 //56
 	"https://chatgpt.com",                       //57
 	"",                                          //58
 	"http://vod.bj.baidubce.com/v3/aigc/vd",     //59
+	"https://vod.bj.baidubce.com",               //60
 }
 
 var ChannelTypeNames = map[int]string{
-	ChannelTypeUnknown:        "Unknown",
-	ChannelTypeOpenAI:         "OpenAI",
-	ChannelTypeMidjourney:     "Midjourney",
-	ChannelTypeAzure:          "Azure",
-	ChannelTypeOllama:         "Ollama",
-	ChannelTypeMidjourneyPlus: "MidjourneyPlus",
-	ChannelTypeOpenAIMax:      "OpenAIMax",
-	ChannelTypeOhMyGPT:        "OhMyGPT",
-	ChannelTypeCustom:         "Custom",
-	ChannelTypeAILS:           "AILS",
-	ChannelTypeAIProxy:        "AIProxy",
-	ChannelTypePaLM:           "PaLM",
-	ChannelTypeAPI2GPT:        "API2GPT",
-	ChannelTypeAIGC2D:         "AIGC2D",
-	ChannelTypeAnthropic:      "Anthropic",
-	ChannelTypeBaidu:          "Baidu",
-	ChannelTypeZhipu:          "Zhipu",
-	ChannelTypeAli:            "Ali",
-	ChannelTypeXunfei:         "Xunfei",
-	ChannelType360:            "360",
-	ChannelTypeOpenRouter:     "OpenRouter",
-	ChannelTypeAIProxyLibrary: "AIProxyLibrary",
-	ChannelTypeFastGPT:        "FastGPT",
-	ChannelTypeTencent:        "Tencent",
-	ChannelTypeGemini:         "Gemini",
-	ChannelTypeMoonshot:       "Moonshot",
-	ChannelTypeZhipu_v4:       "ZhipuV4",
-	ChannelTypePerplexity:     "Perplexity",
-	ChannelTypeLingYiWanWu:    "LingYiWanWu",
-	ChannelTypeAws:            "AWS",
-	ChannelTypeCohere:         "Cohere",
-	ChannelTypeMiniMax:        "MiniMax",
-	ChannelTypeSunoAPI:        "SunoAPI",
-	ChannelTypeDify:           "Dify",
-	ChannelTypeJina:           "Jina",
-	ChannelCloudflare:         "Cloudflare",
-	ChannelTypeSiliconFlow:    "SiliconFlow",
-	ChannelTypeVertexAi:       "VertexAI",
-	ChannelTypeMistral:        "Mistral",
-	ChannelTypeDeepSeek:       "DeepSeek",
-	ChannelTypeMokaAI:         "MokaAI",
-	ChannelTypeVolcEngine:     "VolcEngine",
-	ChannelTypeBaiduV2:        "BaiduV2",
-	ChannelTypeXinference:     "Xinference",
-	ChannelTypeXai:            "xAI",
-	ChannelTypeCoze:           "Coze",
-	ChannelTypeKling:          "Kling",
-	ChannelTypeJimeng:         "Jimeng",
-	ChannelTypeVidu:           "Vidu",
-	ChannelTypeSubmodel:       "Submodel",
-	ChannelTypeDoubaoVideo:    "DoubaoVideo",
-	ChannelTypeSora:           "Sora",
-	ChannelTypeReplicate:      "Replicate",
-	ChannelTypeCodex:          "ChatGPT Subscription (Codex)",
-	ChannelTypeAdvancedCustom: "Advanced Custom",
-	ChannelTypeBaiduVodVidu:   "Baidu VOD Vidu",
+	ChannelTypeUnknown:         "Unknown",
+	ChannelTypeOpenAI:          "OpenAI",
+	ChannelTypeMidjourney:      "Midjourney",
+	ChannelTypeAzure:           "Azure",
+	ChannelTypeOllama:          "Ollama",
+	ChannelTypeMidjourneyPlus:  "MidjourneyPlus",
+	ChannelTypeOpenAIMax:       "OpenAIMax",
+	ChannelTypeOhMyGPT:         "OhMyGPT",
+	ChannelTypeCustom:          "Custom",
+	ChannelTypeAILS:            "AILS",
+	ChannelTypeAIProxy:         "AIProxy",
+	ChannelTypePaLM:            "PaLM",
+	ChannelTypeAPI2GPT:         "API2GPT",
+	ChannelTypeAIGC2D:          "AIGC2D",
+	ChannelTypeAnthropic:       "Anthropic",
+	ChannelTypeBaidu:           "Baidu",
+	ChannelTypeZhipu:           "Zhipu",
+	ChannelTypeAli:             "Ali",
+	ChannelTypeXunfei:          "Xunfei",
+	ChannelType360:             "360",
+	ChannelTypeOpenRouter:      "OpenRouter",
+	ChannelTypeAIProxyLibrary:  "AIProxyLibrary",
+	ChannelTypeFastGPT:         "FastGPT",
+	ChannelTypeTencent:         "Tencent",
+	ChannelTypeGemini:          "Gemini",
+	ChannelTypeMoonshot:        "Moonshot",
+	ChannelTypeZhipu_v4:        "ZhipuV4",
+	ChannelTypePerplexity:      "Perplexity",
+	ChannelTypeLingYiWanWu:     "LingYiWanWu",
+	ChannelTypeAws:             "AWS",
+	ChannelTypeCohere:          "Cohere",
+	ChannelTypeMiniMax:         "MiniMax",
+	ChannelTypeSunoAPI:         "SunoAPI",
+	ChannelTypeDify:            "Dify",
+	ChannelTypeJina:            "Jina",
+	ChannelCloudflare:          "Cloudflare",
+	ChannelTypeSiliconFlow:     "SiliconFlow",
+	ChannelTypeVertexAi:        "VertexAI",
+	ChannelTypeMistral:         "Mistral",
+	ChannelTypeDeepSeek:        "DeepSeek",
+	ChannelTypeMokaAI:          "MokaAI",
+	ChannelTypeVolcEngine:      "VolcEngine",
+	ChannelTypeBaiduV2:         "BaiduV2",
+	ChannelTypeXinference:      "Xinference",
+	ChannelTypeXai:             "xAI",
+	ChannelTypeCoze:            "Coze",
+	ChannelTypeKling:           "Kling",
+	ChannelTypeJimeng:          "Jimeng",
+	ChannelTypeVidu:            "Vidu",
+	ChannelTypeSubmodel:        "Submodel",
+	ChannelTypeDoubaoVideo:     "DoubaoVideo",
+	ChannelTypeSora:            "Sora",
+	ChannelTypeReplicate:       "Replicate",
+	ChannelTypeCodex:           "ChatGPT Subscription (Codex)",
+	ChannelTypeAdvancedCustom:  "Advanced Custom",
+	ChannelTypeBaiduVodVidu:    "Baidu VOD Vidu",
+	ChannelTypeBaiduVodMinimax: "Baidu VOD MiniMax",
 }
 
 func GetChannelTypeName(channelType int) string {
 	if name, ok := ChannelTypeNames[channelType]; ok {
 		return name
diff --git a/controller/model.go b/controller/model.go
index cc2b1eff..cbb101a4 100644
--- a/controller/model.go
+++ b/controller/model.go
@@ -10,10 +10,11 @@ import (
 	"github.com/QuantumNous/new-api/constant"
 	"github.com/QuantumNous/new-api/dto"
 	"github.com/QuantumNous/new-api/model"
 	"github.com/QuantumNous/new-api/relay"
 	"github.com/QuantumNous/new-api/relay/channel/ai360"
+	baidu_vod_minimax "github.com/QuantumNous/new-api/relay/channel/baidu_vod_minimax"
 	"github.com/QuantumNous/new-api/relay/channel/lingyiwanwu"
 	"github.com/QuantumNous/new-api/relay/channel/minimax"
 	"github.com/QuantumNous/new-api/relay/channel/moonshot"
 	relaycommon "github.com/QuantumNous/new-api/relay/common"
 	"github.com/QuantumNous/new-api/relay/helper"
@@ -78,10 +79,18 @@ func init() {
 			Object:  "model",
 			Created: 1626777600,
 			OwnedBy: minimax.ChannelName,
 		})
 	}
+	for _, modelName := range baidu_vod_minimax.ModelList {
+		openAIModels = append(openAIModels, dto.OpenAIModels{
+			Id:      modelName,
+			Object:  "model",
+			Created: 1626777600,
+			OwnedBy: baidu_vod_minimax.ChannelName,
+		})
+	}
 	for modelName, _ := range constant.MidjourneyModel2Action {
 		openAIModels = append(openAIModels, dto.OpenAIModels{
 			Id:      modelName,
 			Object:  "model",
 			Created: 1626777600,
diff --git a/model/pricing_default.go b/model/pricing_default.go
index a58e3f59..79b1b949 100644
--- a/model/pricing_default.go
+++ b/model/pricing_default.go
@@ -33,11 +33,12 @@ var defaultVendorRules = map[string]string{
 	"llama":    "Meta",
 	"doubao":   "瀛楄妭璺冲姩",
 	"kling":    "蹇墜",
 	"jimeng":   "鍗虫ⅵ",
 	"vidu":           "Vidu",
-	"baidu_vod_vidu": "Baidu VOD Vidu",
+	"baidu_vod_vidu":    "Baidu VOD Vidu",
+	"baidu_vod_minimax": "Baidu VOD MiniMax",
 }
 
 // 渚涘簲鍟嗛粯璁ゅ浘鏍囨槧灏? var defaultVendorIcons = map[string]string{
 	"OpenAI":     "OpenAI",
diff --git a/relay/channel/baidu_vod_minimax/adaptor.go b/relay/channel/baidu_vod_minimax/adaptor.go
new file mode 100644
index 00000000..bfc49f91
--- /dev/null
+++ b/relay/channel/baidu_vod_minimax/adaptor.go
@@ -0,0 +1,108 @@
+package baidu_vod_minimax
+
+import (
+	"bytes"
+	"errors"
+	"fmt"
+	"io"
+	"net/http"
+	"strings"
+	"time"
+
+	channelconstant "github.com/QuantumNous/new-api/constant"
+	"github.com/QuantumNous/new-api/dto"
+	"github.com/QuantumNous/new-api/relay/channel"
+	relaycommon "github.com/QuantumNous/new-api/relay/common"
+	"github.com/QuantumNous/new-api/relay/constant"
+	"github.com/QuantumNous/new-api/types"
+
+	"github.com/gin-gonic/gin"
+)
+
+type Adaptor struct {
+}
+
+func (a *Adaptor) ConvertGeminiRequest(*gin.Context, *relaycommon.RelayInfo, *dto.GeminiChatRequest) (any, error) {
+	return nil, errors.New("not implemented")
+}
+
+func (a *Adaptor) ConvertClaudeRequest(*gin.Context, *relaycommon.RelayInfo, *dto.ClaudeRequest) (any, error) {
+	return nil, errors.New("not implemented")
+}
+
+func (a *Adaptor) ConvertAudioRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.AudioRequest) (io.Reader, error) {
+	if info.RelayMode != constant.RelayModeAudioSpeech {
+		return nil, errors.New("unsupported audio relay mode")
+	}
+	raw, outFmt, err := ConvertOpenAIAudioToTTSRequest(info, request)
+	if err != nil {
+		return nil, err
+	}
+	c.Set("response_format", outFmt)
+	return bytes.NewReader(raw), nil
+}
+
+func (a *Adaptor) ConvertImageRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.ImageRequest) (any, error) {
+	return nil, errors.New("not implemented")
+}
+
+func (a *Adaptor) Init(info *relaycommon.RelayInfo) {
+}
+
+func GetRequestURL(info *relaycommon.RelayInfo) (string, error) {
+	base := info.ChannelBaseUrl
+	if base == "" {
+		base = channelconstant.ChannelBaseURLs[channelconstant.ChannelTypeBaiduVodMinimax]
+	}
+	return fmt.Sprintf("%s/v2/tts", strings.TrimRight(base, "/")), nil
+}
+
+func (a *Adaptor) GetRequestURL(info *relaycommon.RelayInfo) (string, error) {
+	return GetRequestURL(info)
+}
+
+func (a *Adaptor) SetupRequestHeader(c *gin.Context, req *http.Header, info *relaycommon.RelayInfo) error {
+	channel.SetupApiRequestHeader(info, c, req)
+	req.Set("Content-Type", "application/json")
+	req.Set("Accept", "application/json")
+	u, err := a.GetRequestURL(info)
+	if err != nil {
+		return err
+	}
+	return ApplyBCEAuth(req, info.ApiKey, http.MethodPost, u, time.Now())
+}
+
+func (a *Adaptor) ConvertOpenAIRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeneralOpenAIRequest) (any, error) {
+	return nil, errors.New("not implemented")
+}
+
+func (a *Adaptor) ConvertRerankRequest(c *gin.Context, relayMode int, request dto.RerankRequest) (any, error) {
+	return nil, errors.New("not implemented")
+}
+
+func (a *Adaptor) ConvertEmbeddingRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.EmbeddingRequest) (any, error) {
+	return nil, errors.New("not implemented")
+}
+
+func (a *Adaptor) ConvertOpenAIResponsesRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.OpenAIResponsesRequest) (any, error) {
+	return nil, errors.New("not implemented")
+}
+
+func (a *Adaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (any, error) {
+	return channel.DoApiRequest(a, c, info, requestBody)
+}
+
+func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (usage any, err *types.NewAPIError) {
+	if info.RelayMode == constant.RelayModeAudioSpeech {
+		return HandleTTSResponse(c, resp, info)
+	}
+	return nil, types.NewError(errors.New("not implemented"), types.ErrorCodeInvalidRequest)
+}
+
+func (a *Adaptor) GetModelList() []string {
+	return ModelList
+}
+
+func (a *Adaptor) GetChannelName() string {
+	return ChannelName
+}
diff --git a/relay/channel/baidu_vod_minimax/adaptor_test.go b/relay/channel/baidu_vod_minimax/adaptor_test.go
new file mode 100644
index 00000000..b007ba01
--- /dev/null
+++ b/relay/channel/baidu_vod_minimax/adaptor_test.go
@@ -0,0 +1,49 @@
+package baidu_vod_minimax
+
+import (
+	"net/http"
+	"net/http/httptest"
+	"strings"
+	"testing"
+
+	relaycommon "github.com/QuantumNous/new-api/relay/common"
+	relayconstant "github.com/QuantumNous/new-api/relay/constant"
+
+	"github.com/gin-gonic/gin"
+	"github.com/stretchr/testify/assert"
+	"github.com/stretchr/testify/require"
+)
+
+func TestGetRequestURL(t *testing.T) {
+	info := &relaycommon.RelayInfo{
+		ChannelMeta: &relaycommon.ChannelMeta{
+			ChannelBaseUrl: "https://vod.bj.baidubce.com",
+		},
+	}
+	u, err := GetRequestURL(info)
+	require.NoError(t, err)
+	assert.Equal(t, "https://vod.bj.baidubce.com/v2/tts", u)
+}
+
+func TestSetupRequestHeaderUsesBCENotBearer(t *testing.T) {
+	gin.SetMode(gin.TestMode)
+	w := httptest.NewRecorder()
+	c, _ := gin.CreateTestContext(w)
+	c.Request = httptest.NewRequest(http.MethodPost, "/v1/audio/speech", nil)
+	c.Request.Header.Set("Content-Type", "application/json")
+
+	info := &relaycommon.RelayInfo{
+		ChannelMeta: &relaycommon.ChannelMeta{
+			ChannelBaseUrl: "https://vod.bj.baidubce.com",
+			ApiKey:         "ak|sk",
+		},
+		RelayMode: relayconstant.RelayModeAudioSpeech,
+	}
+	a := &Adaptor{}
+	h := make(http.Header)
+	require.NoError(t, a.SetupRequestHeader(c, &h, info))
+	assert.Equal(t, "vod.bj.baidubce.com", h.Get("Host"))
+	assert.True(t, strings.HasPrefix(h.Get("Authorization"), "bce-auth-v1/"))
+	assert.NotContains(t, h.Get("Authorization"), "Bearer")
+	assert.Contains(t, h.Get("Authorization"), "/host/")
+}
diff --git a/relay/channel/baidu_vod_minimax/constants.go b/relay/channel/baidu_vod_minimax/constants.go
new file mode 100644
index 00000000..36560ac6
--- /dev/null
+++ b/relay/channel/baidu_vod_minimax/constants.go
@@ -0,0 +1,14 @@
+package baidu_vod_minimax
+
+var ModelList = []string{
+	"speech-2.8-hd",
+	"speech-2.8-turbo",
+	"speech-2.6-hd",
+	"speech-2.6-turbo",
+	"speech-02-hd",
+	"speech-02-turbo",
+	"speech-01-hd",
+	"speech-01-turbo",
+}
+
+var ChannelName = "baidu_vod_minimax"
diff --git a/relay/channel/baidu_vod_minimax/sign.go b/relay/channel/baidu_vod_minimax/sign.go
new file mode 100644
index 00000000..21be7ba0
--- /dev/null
+++ b/relay/channel/baidu_vod_minimax/sign.go
@@ -0,0 +1,135 @@
+package baidu_vod_minimax
+
+import (
+	"crypto/hmac"
+	"crypto/sha256"
+	"encoding/hex"
+	"fmt"
+	"net/http"
+	"net/url"
+	"sort"
+	"strings"
+	"time"
+)
+
+const defaultExpirationSeconds = 1800
+
+func ParseAccessKeys(apiKey string) (string, string, error) {
+	parts := strings.Split(apiKey, "|")
+	if len(parts) != 2 {
+		return "", "", fmt.Errorf("invalid api_key, required format is accessKey|secretKey")
+	}
+	ak := strings.TrimSpace(parts[0])
+	sk := strings.TrimSpace(parts[1])
+	if ak == "" || sk == "" {
+		return "", "", fmt.Errorf("invalid api_key, required format is accessKey|secretKey")
+	}
+	return ak, sk, nil
+}
+
+func uriEncode(s string, encodeSlash bool) string {
+	var b strings.Builder
+	for i := 0; i < len(s); i++ {
+		c := s[i]
+		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') ||
+			c == '_' || c == '-' || c == '~' || c == '.' {
+			b.WriteByte(c)
+		} else if c == '/' {
+			if encodeSlash {
+				b.WriteString("%2F")
+			} else {
+				b.WriteByte(c)
+			}
+		} else {
+			b.WriteString(fmt.Sprintf("%%%02X", c))
+		}
+	}
+	return b.String()
+}
+
+func hmacSHA256Hex(key []byte, msg string) string {
+	m := hmac.New(sha256.New, key)
+	m.Write([]byte(msg))
+	return hex.EncodeToString(m.Sum(nil))
+}
+
+func SignAuthorization(ak, sk, method, host, canonicalURI, canonicalQuery string, now time.Time, expirationSec int) (string, error) {
+	if expirationSec <= 0 {
+		expirationSec = defaultExpirationSeconds
+	}
+	host = strings.TrimSpace(host)
+	if canonicalURI == "" {
+		canonicalURI = "/"
+	}
+	if !strings.HasPrefix(canonicalURI, "/") {
+		canonicalURI = "/" + canonicalURI
+	}
+	timestamp := now.UTC().Format("2006-01-02T15:04:05Z")
+	authPrefix := fmt.Sprintf("bce-auth-v1/%s/%s/%d", ak, timestamp, expirationSec)
+	canonicalURIEnc := uriEncode(canonicalURI, false)
+	canonicalHeaders := "host:" + uriEncode(host, true)
+	canonicalRequest := strings.ToUpper(method) + "\n" +
+		canonicalURIEnc + "\n" +
+		canonicalQuery + "\n" +
+		canonicalHeaders
+	signingKey := hmacSHA256Hex([]byte(sk), authPrefix)
+	signature := hmacSHA256Hex([]byte(signingKey), canonicalRequest)
+	return fmt.Sprintf("%s/host/%s", authPrefix, signature), nil
+}
+
+func ApplyBCEAuth(header *http.Header, apiKey, method, requestURL string, now time.Time) error {
+	if header == nil {
+		return fmt.Errorf("header is nil")
+	}
+	ak, sk, err := ParseAccessKeys(apiKey)
+	if err != nil {
+		return err
+	}
+	u, err := url.Parse(requestURL)
+	if err != nil {
+		return fmt.Errorf("parse request url: %w", err)
+	}
+	host := strings.TrimSpace(u.Host)
+	if host == "" {
+		return fmt.Errorf("request url missing host")
+	}
+	path := u.Path
+	if path == "" {
+		path = "/"
+	}
+	canonicalQuery := canonicalizeQuery(u.Query())
+	auth, err := SignAuthorization(ak, sk, method, host, path, canonicalQuery, now, defaultExpirationSeconds)
+	if err != nil {
+		return err
+	}
+	header.Set("Host", host)
+	header.Set("Authorization", auth)
+	return nil
+}
+
+func canonicalizeQuery(values url.Values) string {
+	if len(values) == 0 {
+		return ""
+	}
+	keys := make([]string, 0, len(values))
+	for k := range values {
+		if strings.EqualFold(k, "authorization") {
+			continue
+		}
+		keys = append(keys, k)
+	}
+	sort.Strings(keys)
+	parts := make([]string, 0, len(keys))
+	for _, k := range keys {
+		vs := values[k]
+		if len(vs) == 0 {
+			parts = append(parts, uriEncode(k, true)+"=")
+			continue
+		}
+		for _, v := range vs {
+			parts = append(parts, uriEncode(k, true)+"="+uriEncode(v, true))
+		}
+	}
+	sort.Strings(parts)
+	return strings.Join(parts, "&")
+}
diff --git a/relay/channel/baidu_vod_minimax/sign_test.go b/relay/channel/baidu_vod_minimax/sign_test.go
new file mode 100644
index 00000000..35502c12
--- /dev/null
+++ b/relay/channel/baidu_vod_minimax/sign_test.go
@@ -0,0 +1,103 @@
+package baidu_vod_minimax
+
+import (
+	"crypto/hmac"
+	"crypto/sha256"
+	"encoding/hex"
+	"fmt"
+	"net/http"
+	"strings"
+	"testing"
+	"time"
+
+	"github.com/stretchr/testify/assert"
+	"github.com/stretchr/testify/require"
+)
+
+const goldenAuthorization = "bce-auth-v1/ak-test/2026-06-12T02:45:13Z/1800/host/b6952868b8ae3da6a73cd732e90d620f23f6ae3ecce40832c97fdcd729f8902f"
+
+func TestParseAccessKeys(t *testing.T) {
+	ak, sk, err := ParseAccessKeys("ak123|sk456")
+	require.NoError(t, err)
+	assert.Equal(t, "ak123", ak)
+	assert.Equal(t, "sk456", sk)
+
+	_, _, err = ParseAccessKeys("only-one")
+	require.Error(t, err)
+}
+
+func TestSignAuthorizationGoldenVector(t *testing.T) {
+	now := time.Date(2026, 6, 12, 2, 45, 13, 0, time.UTC)
+	auth, err := SignAuthorization(
+		"ak-test", "sk-test",
+		http.MethodPost, "vod.bj.baidubce.com", "/v2/tts", "",
+		now, 1800,
+	)
+	require.NoError(t, err)
+
+	authPrefix := "bce-auth-v1/ak-test/2026-06-12T02:45:13Z/1800"
+	canonicalRequest := "POST\n/v2/tts\n\nhost:vod.bj.baidubce.com"
+	signingKeyMAC := hmac.New(sha256.New, []byte("sk-test"))
+	signingKeyMAC.Write([]byte(authPrefix))
+	signingKey := hex.EncodeToString(signingKeyMAC.Sum(nil))
+	sigMAC := hmac.New(sha256.New, []byte(signingKey))
+	sigMAC.Write([]byte(canonicalRequest))
+	expectedSig := hex.EncodeToString(sigMAC.Sum(nil))
+	expectedAuth := fmt.Sprintf("%s/host/%s", authPrefix, expectedSig)
+
+	assert.Equal(t, expectedAuth, auth)
+	assert.Equal(t, goldenAuthorization, auth)
+}
+
+func TestSignAuthorizationTrimsHost(t *testing.T) {
+	now := time.Date(2026, 6, 12, 2, 45, 13, 0, time.UTC)
+	withSpace, err := SignAuthorization(
+		"ak-test", "sk-test",
+		http.MethodPost, "  vod.bj.baidubce.com  ", "/v2/tts", "",
+		now, 1800,
+	)
+	require.NoError(t, err)
+	withoutSpace, err := SignAuthorization(
+		"ak-test", "sk-test",
+		http.MethodPost, "vod.bj.baidubce.com", "/v2/tts", "",
+		now, 1800,
+	)
+	require.NoError(t, err)
+	assert.Equal(t, withoutSpace, withSpace)
+	assert.Equal(t, goldenAuthorization, withSpace)
+}
+
+func TestSignAuthorizationContainsHostSignedHeader(t *testing.T) {
+	now := time.Date(2026, 6, 12, 2, 45, 13, 0, time.UTC)
+	auth, err := SignAuthorization(
+		"ak-test", "sk-test",
+		http.MethodPost, "vod.bj.baidubce.com", "/v2/tts", "",
+		now, 1800,
+	)
+	require.NoError(t, err)
+	assert.True(t, strings.HasPrefix(auth, "bce-auth-v1/ak-test/2026-06-12T02:45:13Z/1800/host/"))
+	assert.NotContains(t, auth, "Bearer")
+	parts := strings.Split(auth, "/")
+	require.Len(t, parts, 6)
+	assert.Equal(t, "host", parts[4])
+	assert.Regexp(t, "^[0-9a-f]{64}$", parts[5])
+}
+
+func TestSignAuthorizationDeterministic(t *testing.T) {
+	now := time.Date(2026, 6, 12, 2, 45, 13, 0, time.UTC)
+	a, err := SignAuthorization("ak", "sk", http.MethodPost, "vod.bj.baidubce.com", "/v2/tts", "", now, 1800)
+	require.NoError(t, err)
+	b, err := SignAuthorization("ak", "sk", http.MethodPost, "vod.bj.baidubce.com", "/v2/tts", "", now, 1800)
+	require.NoError(t, err)
+	assert.Equal(t, a, b)
+}
+
+func TestApplyBCEAuthSetsHostAndAuthorization(t *testing.T) {
+	h := make(http.Header)
+	now := time.Date(2026, 6, 12, 2, 45, 13, 0, time.UTC)
+	err := ApplyBCEAuth(&h, "ak|sk", http.MethodPost, "https://vod.bj.baidubce.com/v2/tts", now)
+	require.NoError(t, err)
+	assert.Equal(t, "vod.bj.baidubce.com", h.Get("Host"))
+	assert.True(t, strings.HasPrefix(h.Get("Authorization"), "bce-auth-v1/"))
+	assert.Contains(t, h.Get("Authorization"), "/host/")
+}
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
diff --git a/relay/relay_adaptor.go b/relay/relay_adaptor.go
index 198bbc29..0282caf6 100644
--- a/relay/relay_adaptor.go
+++ b/relay/relay_adaptor.go
@@ -8,10 +8,11 @@ import (
 	"github.com/QuantumNous/new-api/relay/channel/advancedcustom"
 	"github.com/QuantumNous/new-api/relay/channel/ali"
 	"github.com/QuantumNous/new-api/relay/channel/aws"
 	"github.com/QuantumNous/new-api/relay/channel/baidu"
 	"github.com/QuantumNous/new-api/relay/channel/baidu_v2"
+	baidu_vod_minimax "github.com/QuantumNous/new-api/relay/channel/baidu_vod_minimax"
 	"github.com/QuantumNous/new-api/relay/channel/claude"
 	"github.com/QuantumNous/new-api/relay/channel/cloudflare"
 	"github.com/QuantumNous/new-api/relay/channel/codex"
 	"github.com/QuantumNous/new-api/relay/channel/cohere"
 	"github.com/QuantumNous/new-api/relay/channel/coze"
@@ -30,20 +31,20 @@ import (
 	"github.com/QuantumNous/new-api/relay/channel/perplexity"
 	"github.com/QuantumNous/new-api/relay/channel/replicate"
 	"github.com/QuantumNous/new-api/relay/channel/siliconflow"
 	"github.com/QuantumNous/new-api/relay/channel/submodel"
 	taskali "github.com/QuantumNous/new-api/relay/channel/task/ali"
+	taskBaiduVodVidu "github.com/QuantumNous/new-api/relay/channel/task/baidu_vod_vidu"
 	taskdoubao "github.com/QuantumNous/new-api/relay/channel/task/doubao"
 	taskGemini "github.com/QuantumNous/new-api/relay/channel/task/gemini"
 	"github.com/QuantumNous/new-api/relay/channel/task/hailuo"
 	taskjimeng "github.com/QuantumNous/new-api/relay/channel/task/jimeng"
 	"github.com/QuantumNous/new-api/relay/channel/task/kling"
 	tasksora "github.com/QuantumNous/new-api/relay/channel/task/sora"
 	"github.com/QuantumNous/new-api/relay/channel/task/suno"
 	taskvertex "github.com/QuantumNous/new-api/relay/channel/task/vertex"
 	taskVidu "github.com/QuantumNous/new-api/relay/channel/task/vidu"
-	taskBaiduVodVidu "github.com/QuantumNous/new-api/relay/channel/task/baidu_vod_vidu"
 	"github.com/QuantumNous/new-api/relay/channel/tencent"
 	"github.com/QuantumNous/new-api/relay/channel/vertex"
 	"github.com/QuantumNous/new-api/relay/channel/volcengine"
 	"github.com/QuantumNous/new-api/relay/channel/xai"
 	"github.com/QuantumNous/new-api/relay/channel/xunfei"
@@ -122,10 +123,12 @@ func GetAdaptor(apiType int) channel.Adaptor {
 		return &replicate.Adaptor{}
 	case constant.APITypeCodex:
 		return &codex.Adaptor{}
 	case constant.APITypeAdvancedCustom:
 		return &advancedcustom.Adaptor{}
+	case constant.APITypeBaiduVodMinimax:
+		return &baidu_vod_minimax.Adaptor{}
 	}
 	return nil
 }
 
 func GetTaskPlatform(c *gin.Context) constant.TaskPlatform {
diff --git a/web/classic/src/constants/channel.constants.js b/web/classic/src/constants/channel.constants.js
index 4337b53e..9d83751a 100644
--- a/web/classic/src/constants/channel.constants.js
+++ b/web/classic/src/constants/channel.constants.js
@@ -167,10 +167,15 @@ export const CHANNEL_OPTIONS = [
   {
     value: 59,
     color: 'blue',
     label: 'Baidu VOD Vidu',
   },
+  {
+    value: 60,
+    color: 'blue',
+    label: 'Baidu VOD MiniMax',
+  },
   {
     value: 53,
     color: 'blue',
     label: 'SubModel',
   },
diff --git a/web/default/src/features/channels/constants.ts b/web/default/src/features/channels/constants.ts
index 7a58d8fd..544cf45b 100644
--- a/web/default/src/features/channels/constants.ts
+++ b/web/default/src/features/channels/constants.ts
@@ -76,16 +76,17 @@ export const CHANNEL_TYPES = {
   55: 'Sora',
   56: 'Replicate',
   57: 'ChatGPT Subscription (Codex)',
   58: 'Advanced Custom',
   59: 'Baidu VOD Vidu',
+  60: 'Baidu VOD MiniMax',
 } as const
 
 const CHANNEL_TYPE_DISPLAY_ORDER: number[] = [
   1, 14, 33, 24, 43, 3, 41, 48, 58, 42, 34, 20, 4, 40, 27, 25, 17, 26, 15, 46,
   23, 18, 45, 31, 35, 49, 19, 47, 37, 38, 39, 11, 8, 57, 22, 21, 44, 2, 5, 36,
-  50, 51, 52, 59, 53, 54, 55, 56,
+  50, 51, 52, 59, 60, 53, 54, 55, 56,
 ]
 
 export const CHANNEL_TYPE_OPTIONS: { value: number; label: string }[] = (() => {
   const ordered: { value: number; label: string }[] = []
   const seen = new Set<number>()
diff --git a/web/default/src/features/channels/lib/channel-utils.ts b/web/default/src/features/channels/lib/channel-utils.ts
index 69bfc5f1..027253be 100644
--- a/web/default/src/features/channels/lib/channel-utils.ts
+++ b/web/default/src/features/channels/lib/channel-utils.ts
@@ -97,10 +97,11 @@ export function getChannelTypeIcon(type: number): string {
     5: 'Midjourney', // MjProxyPlus
     50: 'Kling', // Kling
     51: 'Jimeng', // Jimeng
     52: 'Vidu', // Vidu
     59: 'Baidu', // Baidu VOD Vidu
+    60: 'Baidu', // Baidu VOD MiniMax
     36: 'Suno', // SunoAPI
     55: 'OpenAI', // Sora
     54: 'Doubao', // DoubaoVideo
     56: 'Replicate', // Replicate
 

```
