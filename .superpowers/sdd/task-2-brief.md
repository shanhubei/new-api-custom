### Task 2: Channel / APIType wiring

**Files:**
- Modify: `constant/channel.go` 鈥?insert type 60 before Dummy; append base URL; add name
- Modify: `constant/api_type.go` 鈥?insert `APITypeBaiduVodMinimax` before Dummy
- Modify: `common/api_type.go` 鈥?add case
- Modify: `relay/relay_adaptor.go` 鈥?import + GetAdaptor case
- Create: `relay/channel/baidu_vod_minimax/constants.go`
- Create: `relay/channel/baidu_vod_minimax/adaptor.go` (stub compiling)

**Interfaces:**
- Consumes: signer from Task 1
- Produces: `ChannelTypeBaiduVodMinimax = 60`, `APITypeBaiduVodMinimax`, `GetAdaptor` returns `*baidu_vod_minimax.Adaptor`

- [ ] **Step 1: Add constants**

In `constant/channel.go`:

```go
ChannelTypeBaiduVodVidu    = 59
ChannelTypeBaiduVodMinimax = 60
ChannelTypeDummy           // this one is only for count, do not add any channel after this
```

Append to `ChannelBaseURLs` after index 59 entry:

```go
"http://vod.bj.baidubce.com/v3/aigc/vd", //59
"https://vod.bj.baidubce.com",           //60
```

Add:

```go
ChannelTypeBaiduVodMinimax: "Baidu VOD MiniMax",
```

In `constant/api_type.go` before Dummy:

```go
APITypeAdvancedCustom
APITypeBaiduVodMinimax
APITypeDummy
```

In `common/api_type.go`:

```go
case constant.ChannelTypeBaiduVodMinimax:
	apiType = constant.APITypeBaiduVodMinimax
```

- [ ] **Step 2: Create `constants.go` + stub `adaptor.go`**

```go
package baidu_vod_minimax

var ModelList = []string{
	"speech-2.8-hd",
	"speech-2.8-turbo",
	"speech-2.6-hd",
	"speech-2.6-turbo",
	"speech-02-hd",
	"speech-02-turbo",
	"speech-01-hd",
	"speech-01-turbo",
}

var ChannelName = "baidu_vod_minimax"
```

Stub adaptor must implement `channel.Adaptor` enough to compile: copy method set from `relay/channel/minimax/adaptor.go`, but:

- `SetupRequestHeader`: call `channel.SetupApiRequestHeader`, then `ApplyBCEAuth(req, info.ApiKey, c.Request.Method /* wrong */, requestURL)` 鈥?actually method should be POST and URL from `GetRequestURL(info)`. Use:

```go
func (a *Adaptor) SetupRequestHeader(c *gin.Context, req *http.Header, info *relaycommon.RelayInfo) error {
	channel.SetupApiRequestHeader(info, c, req)
	req.Set("Content-Type", "application/json")
	req.Set("Accept", "application/json")
	u, err := a.GetRequestURL(info)
	if err != nil {
		return err
	}
	return ApplyBCEAuth(req, info.ApiKey, http.MethodPost, u, time.Now())
}
```

- `GetRequestURL`:

```go
func GetRequestURL(info *relaycommon.RelayInfo) (string, error) {
	base := info.ChannelBaseUrl
	if base == "" {
		base = channelconstant.ChannelBaseURLs[channelconstant.ChannelTypeBaiduVodMinimax]
	}
	return fmt.Sprintf("%s/v2/tts", strings.TrimRight(base, "/")), nil
}
```

- Non-speech convert methods: `return nil, errors.New("not implemented")`
- `DoResponse`: for now return error unless audio speech (filled in Task 3/4)

Wire `relay/relay_adaptor.go`:

```go
baidu_vod_minimax "github.com/QuantumNous/new-api/relay/channel/baidu_vod_minimax"
...
case constant.APITypeBaiduVodMinimax:
	return &baidu_vod_minimax.Adaptor{}
```

- [ ] **Step 3: Compile check**

```bash
go test ./relay/channel/baidu_vod_minimax/ ./common/ ./relay/ -count=1 -run '^$'
```

Expected: compile OK (or only existing failures unrelated)

- [ ] **Step 4: Write adaptor URL test**

```go
func TestGetRequestURL(t *testing.T) {
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl: "https://vod.bj.baidubce.com",
		},
	}
	u, err := GetRequestURL(info)
	require.NoError(t, err)
	assert.Equal(t, "https://vod.bj.baidubce.com/v2/tts", u)
}
```

- [ ] **Step 5: Commit** (if requested)

```bash
git add constant/channel.go constant/api_type.go common/api_type.go relay/relay_adaptor.go relay/channel/baidu_vod_minimax/
git commit -m "feat(baidu-vod-minimax): wire channel type 60 and adaptor stub"
```

---

