### Task 6: Final verification

- [ ] **Step 1: Full package tests**

```bash
go test ./relay/channel/baidu_vod_minimax/ -count=1
```

Expected: all PASS

- [ ] **Step 2: Ensure ChannelBaseURLs length matches Dummy**

Quick check: `len(ChannelBaseURLs)` must equal `ChannelTypeDummy` (index of Dummy). After adding 60, array must have 61 entries (0..60).

- [ ] **Step 3: Manual checklist (document in PR / notes)**

1. Admin create channel type Baidu VOD MiniMax, Base empty or `https://vod.bj.baidubce.com`, Key `AK|SK`
2. Enable model `speech-2.8-hd` on that channel only (not same group as official MiniMax same name)
3. `POST /v1/audio/speech` with that model 鈫?audio URL or bytes
4. Official MiniMax + Baidu VOD Vidu still work

---

## Spec coverage self-review

| Spec requirement | Task |
|------------------|------|
| Channel type 60 + name + default Base | Task 2 |
| APIType + GetAdaptor | Task 2 |
| `bce-auth-v1` + `host` + `AK\|SK` | Task 1, 4 |
| `/v2/tts` URL | Task 2 |
| OpenAI speech 鈫?MiniMax body | Task 3 |
| Response url/hex + usage_characters | Task 3 |
| Model list speech-* | Task 2/4 |
| Frontend selectable | Task 5 |
| No change MiniMax/Vidu | All tasks avoid those packages except copy patterns |
| Tests | Tasks 1鈥?, 6 |

**Pinned decisions:** `expirationSeconds = 1800`; signedHeaders = `host` only; phase-1 TTS only.

---

## Execution Handoff

Plan saved to `docs/superpowers/plans/2026-08-26-baidu-vod-minimax-tts.md`.

**Two execution options:**

1. **Subagent-Driven (recommended)** 鈥?fresh subagent per task, review between tasks  
2. **Inline Execution** 鈥?implement tasks in this session with checkpoints  

Which approach?
