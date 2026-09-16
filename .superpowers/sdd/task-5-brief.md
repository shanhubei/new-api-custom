### Task 5: Frontend channel type

**Files:**
- Modify: `web/default/src/features/channels/constants.ts` 鈥?`60: 'Baidu VOD MiniMax'`
- Modify: `web/default/src/features/channels/lib/channel-utils.ts` 鈥?icon map `60: 'Baidu'` (same as 59)
- Modify: `web/classic/src/constants/channel.constants.js` 鈥?option value 60

- [ ] **Step 1: Add UI entries** next to Baidu VOD Vidu (59)

default `constants.ts`:

```ts
60: 'Baidu VOD MiniMax',
```

`channel-utils.ts`:

```ts
60: 'Baidu', // Baidu VOD MiniMax
```

classic:

```js
{
  value: 60,
  color: 'blue',
  label: 'Baidu VOD MiniMax',
},
```

- [ ] **Step 2: Sanity 鈥?no TS need if constants-only; skip build unless required**

- [ ] **Step 3: Commit** (if requested)

```bash
git commit -am "feat(baidu-vod-minimax): expose channel type in admin UI"
```

---

