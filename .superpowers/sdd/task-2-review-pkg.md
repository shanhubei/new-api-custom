# Review Package Task 2
BASE: 594502befdca34aee10741823dbe11be9615c519
HEAD: 9cba93a3d7ee19bcf56ee19341895872313f8c63

## Commits
9cba93a3 feat(baidu-vod-minimax): wire channel type 60 and adaptor stub

## Stat
 common/api_type.go                              |   2 +
 constant/api_type.go                            |   1 +
 constant/channel.go                             | 229 ++++++++++++------------
 relay/channel/baidu_vod_minimax/adaptor.go      |  95 ++++++++++
 relay/channel/baidu_vod_minimax/adaptor_test.go |  21 +++
 relay/channel/baidu_vod_minimax/constants.go    |  14 ++
 relay/relay_adaptor.go                          |   5 +-
 7 files changed, 253 insertions(+), 114 deletions(-)

## Diff
```
diff --git a/common/api_type.go b/common/api_type.go
index c198ffc0..549e2fff 100644
--- a/common/api_type.go
+++ b/common/api_type.go
@@ -72,14 +72,16 @@ func ChannelType2APIType(channelType int) (int, bool) {
 	case constant.ChannelTypeMiniMax:
 		apiType = constant.APITypeMiniMax
 	case constant.ChannelTypeReplicate:
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
 }
diff --git a/constant/api_type.go b/constant/api_type.go
index f3657a11..90116ca7 100644
--- a/constant/api_type.go
+++ b/constant/api_type.go
@@ -32,10 +32,11 @@ const (
 	APITypeCoze
 	APITypeJimeng
 	APITypeMoonshot
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
@@ -1,68 +1,69 @@
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
 	"https://api.openai.com",              // 1
 	"https://oa.api2d.net",                // 2
 	"",                                    // 3
@@ -117,75 +118,77 @@ var ChannelBaseURLs = []string{
 	"https://api.vidu.cn",                       //52
 	"https://llm.submodel.ai",                   //53
 	"https://ark.cn-beijing.volces.com",         //54
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
 	}
 	return "Unknown"
 }
diff --git a/relay/channel/baidu_vod_minimax/adaptor.go b/relay/channel/baidu_vod_minimax/adaptor.go
new file mode 100644
index 00000000..70ef846f
--- /dev/null
+++ b/relay/channel/baidu_vod_minimax/adaptor.go
@@ -0,0 +1,95 @@
+package baidu_vod_minimax
+
+import (
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
+	return nil, errors.New("not implemented")
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
index 00000000..0cfc7734
--- /dev/null
+++ b/relay/channel/baidu_vod_minimax/adaptor_test.go
@@ -0,0 +1,21 @@
+package baidu_vod_minimax
+
+import (
+	"testing"
+
+	relaycommon "github.com/QuantumNous/new-api/relay/common"
+
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
diff --git a/relay/relay_adaptor.go b/relay/relay_adaptor.go
index 198bbc29..0282caf6 100644
--- a/relay/relay_adaptor.go
+++ b/relay/relay_adaptor.go
@@ -5,16 +5,17 @@ import (
 
 	"github.com/QuantumNous/new-api/constant"
 	"github.com/QuantumNous/new-api/relay/channel"
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
 	"github.com/QuantumNous/new-api/relay/channel/deepseek"
 	"github.com/QuantumNous/new-api/relay/channel/dify"
 	"github.com/QuantumNous/new-api/relay/channel/gemini"
@@ -27,26 +28,26 @@ import (
 	"github.com/QuantumNous/new-api/relay/channel/ollama"
 	"github.com/QuantumNous/new-api/relay/channel/openai"
 	"github.com/QuantumNous/new-api/relay/channel/palm"
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
 	"github.com/QuantumNous/new-api/relay/channel/zhipu"
 	"github.com/QuantumNous/new-api/relay/channel/zhipu_4v"
 	"github.com/gin-gonic/gin"
@@ -119,16 +120,18 @@ func GetAdaptor(apiType int) channel.Adaptor {
 	case constant.APITypeMiniMax:
 		return &minimax.Adaptor{}
 	case constant.APITypeReplicate:
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
 	channelType := c.GetInt("channel_type")
 	if channelType > 0 {
 		return constant.TaskPlatform(strconv.Itoa(channelType))

```
