# Task 5 Report: Frontend channel type for Baidu VOD MiniMax

## Status

**Complete**

## Changes

| File | Change |
|------|--------|
| `web/default/src/features/channels/constants.ts` | Added `60: 'Baidu VOD MiniMax'` to `CHANNEL_TYPES`; added `60` to `CHANNEL_TYPE_DISPLAY_ORDER` after `59` |
| `web/default/src/features/channels/lib/channel-utils.ts` | Added icon map entry `60: 'Baidu'` (same as type 59) |
| `web/classic/src/constants/channel.constants.js` | Added option `{ value: 60, color: 'blue', label: 'Baidu VOD MiniMax' }` after type 59 |

## Verification

- Constants-only change; no new TypeScript types required.
- Build skipped per brief (sanity check only).

## Commit

```
50eb43f6 feat(baidu-vod-minimax): expose channel type in admin UI
```
