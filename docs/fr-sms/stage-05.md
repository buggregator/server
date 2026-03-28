# Stage 5: Frontend — Preview Card + Detail Page

## Overview

Build the two UI components that display SMS events: a compact `preview-card` shown in the
events list, and a `sms-page` shown when the user opens an event. Both are intentionally simple
— SMS events have very few fields. The preview shows `from → to` + message snippet (same pill
pattern as SMTP). The detail page shows all fields plus a colored gateway badge.

## Files

CREATE:
- `frontend/src/entities/sms/ui/preview-card/preview-card.vue`
- `frontend/src/entities/sms/ui/preview-card/index.ts`
- `frontend/src/entities/sms/ui/sms-page/sms-page.vue`
- `frontend/src/entities/sms/ui/sms-page/index.ts`
- `frontend/src/entities/sms/ui/index.ts`

MODIFY:
- `frontend/src/widgets/ui/event-card-mapper/event-card-mapper.vue` — replace placeholder with `PreviewCard`
- `frontend/src/widgets/ui/event-page-mapper/event-page-mapper.vue` — replace placeholder with `SmsPage`
- `frontend/src/entities/sms/index.ts` — ensure `ui` exports are included

## Code References

- `frontend/src/entities/smtp/ui/preview-card/preview-card.vue` — copy the pill flow layout
  (`from → to`), text preview, and Tailwind classes. Replace `subject` with `message` preview.
- `frontend/src/entities/smtp/ui/smtp-page/smtp-page.vue` — detail page structure pattern
- `frontend/src/shared/ui/event-detail-layout/` — layout wrapper for detail pages
- `frontend/src/widgets/ui/event-card-mapper/event-card-mapper.vue` — where to swap placeholder

## Implementation Details

### preview-card.vue

Show:
- `from → to` pill row (same SVG arrow, same pill classes as SMTP)
- Message text preview (trimmed to 120 chars, `line-clamp-2`)
- Gateway badge (e.g. `twilio` in small monospace badge)

```vue
<script lang="ts" setup>
import { computed } from 'vue'
import type { NormalizedEvent } from '@/shared/types'
import { PreviewCard } from '@/shared/ui'
import type { SmsPayload } from '../../types'

type Props = { event: NormalizedEvent<SmsPayload> }
const props = defineProps<Props>()

const eventLink = computed(() => `/sms/${props.event.id}`)
const textPreview = computed(() => {
  const t = props.event.payload.message || ''
  return t.length > 120 ? `${t.substring(0, 120)}...` : t
})
</script>
```

### sms-page.vue

Show full detail:
- Header: gateway badge (colored by name: twilio=red, vonage=blue, plivo=green, generic=gray)
- Metadata table: From, To
- Message body (plain text, pre-wrap)

Use the shared `event-detail-layout` wrapper if available (check SMTP page for reference).

### Gateway badge colors

```typescript
const GATEWAY_COLORS: Record<string, string> = {
  twilio:  'bg-red-100 dark:bg-red-500/15 text-red-700 dark:text-red-400',
  vonage:  'bg-blue-100 dark:bg-blue-500/15 text-blue-700 dark:text-blue-400',
  plivo:   'bg-green-100 dark:bg-green-500/15 text-green-700 dark:text-green-400',
  generic: 'bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-300',
}
```

## Definition of Done

- [ ] `preview-card.vue` renders `from → to` pills and message text snippet
- [ ] `preview-card.vue` shows gateway name as a small badge
- [ ] `preview-card.vue` links to `/sms/:id`
- [ ] `sms-page.vue` displays `from`, `to`, `gateway` badge, and full `message`
- [ ] Both components use `NormalizedEvent<SmsPayload>` props
- [ ] `event-card-mapper.vue` uses real `PreviewCard` (not placeholder)
- [ ] `event-page-mapper.vue` uses real `SmsPage` (not placeholder)
- [ ] No TypeScript errors

## Dependencies

**Requires**: Stage 4 (types, normalizer, mapper entries)
**Enables**: Stage 6 (final registration and docs)
