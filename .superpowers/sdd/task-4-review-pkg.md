# Review Package Task 4
BASE: aad9917cfe9feceb54fffacec2fcc8f3ead84086
HEAD: 21f97bd33192bbd870ac00d1160fe7d1190c95fd

## Commits
21f97bd3 feat(baidu-vod-minimax): register models and header smoke test

## Stat
 controller/model.go                             |  9 ++++++++
 model/pricing_default.go                        |  3 ++-
 relay/channel/baidu_vod_minimax/adaptor_test.go | 28 +++++++++++++++++++++++++
 3 files changed, 39 insertions(+), 1 deletion(-)

## Diff
```
diff --git a/controller/model.go b/controller/model.go
index cc2b1eff..cbb101a4 100644
--- a/controller/model.go
+++ b/controller/model.go
@@ -9,12 +9,13 @@ import (
 	"github.com/QuantumNous/new-api/common"
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
 	"github.com/QuantumNous/new-api/service"
@@ -77,12 +78,20 @@ func init() {
 			Id:      modelName,
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
 			OwnedBy: "midjourney",
diff --git a/model/pricing_default.go b/model/pricing_default.go
index a58e3f59..79b1b949 100644
--- a/model/pricing_default.go
+++ b/model/pricing_default.go
@@ -32,13 +32,14 @@ var defaultVendorRules = map[string]string{
 	"grok":     "xAI",
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
 	"Anthropic":  "Claude.Color",
diff --git a/relay/channel/baidu_vod_minimax/adaptor_test.go b/relay/channel/baidu_vod_minimax/adaptor_test.go
index 0cfc7734..b007ba01 100644
--- a/relay/channel/baidu_vod_minimax/adaptor_test.go
+++ b/relay/channel/baidu_vod_minimax/adaptor_test.go
@@ -1,13 +1,18 @@
 package baidu_vod_minimax
 
 import (
+	"net/http"
+	"net/http/httptest"
+	"strings"
 	"testing"
 
 	relaycommon "github.com/QuantumNous/new-api/relay/common"
+	relayconstant "github.com/QuantumNous/new-api/relay/constant"
 
+	"github.com/gin-gonic/gin"
 	"github.com/stretchr/testify/assert"
 	"github.com/stretchr/testify/require"
 )
 
 func TestGetRequestURL(t *testing.T) {
 	info := &relaycommon.RelayInfo{
@@ -16,6 +21,29 @@ func TestGetRequestURL(t *testing.T) {
 		},
 	}
 	u, err := GetRequestURL(info)
 	require.NoError(t, err)
 	assert.Equal(t, "https://vod.bj.baidubce.com/v2/tts", u)
 }
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

```
