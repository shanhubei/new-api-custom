# Review Package Task 5
BASE: 21f97bd33192bbd870ac00d1160fe7d1190c95fd
HEAD: 50eb43f661ef3eac4762b418f8b2461b03090a27

## Commits
50eb43f6 feat(baidu-vod-minimax): expose channel type in admin UI

## Stat
 web/classic/src/constants/channel.constants.js         | 5 +++++
 web/default/src/features/channels/constants.ts         | 3 ++-
 web/default/src/features/channels/lib/channel-utils.ts | 1 +
 3 files changed, 8 insertions(+), 1 deletion(-)

## Diff
```
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
