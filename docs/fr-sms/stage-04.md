# Stage 4: Frontend — Types + Normalizer

## Overview

Add the `sms` event type to the frontend type system and create the entity's data layer:
TypeScript interface for the payload, a normalizer function, and all barrel exports.
No UI yet — just the plumbing so the mappers in `event-card-mapper` and `event-page-mapper`
can reference the SMS entity.

## Files

MODIFY:
- `frontend/src/shared/types/events.ts` — add `Sms = 'sms'` to `EventTypes` enum

CREATE:
- `frontend/src/entities/sms/types.ts` — `SmsPayload` interface
- `frontend/src/entities/sms/lib/normalize-sms-event.ts` — normalizer
- `frontend/src/entities/sms/lib/index.ts` — barrel export
- `frontend/src/entities/sms/index.ts` — top-level barrel
- `frontend/src/entities/sms/mocks/sms-twilio.json` — sample payload for stories

MODIFY (placeholders for Stage 5 — add empty imports so linting passes):
- `frontend/src/widgets/ui/event-card-mapper/event-card-mapper.vue` — import + register
- `frontend/src/widgets/ui/event-page-mapper/event-page-mapper.vue` — import + register

## Code References

- `frontend/src/shared/types/events.ts` — add `Sms = 'sms'` alongside existing entries
- `frontend/src/entities/smtp/types.ts` — pattern for payload interface
- `frontend/src/entities/smtp/lib/normalize-smtp-event.ts` — exact pattern to follow
- `frontend/src/entities/smtp/index.ts` — barrel export pattern
- `frontend/src/widgets/ui/event-card-mapper/event-card-mapper.vue` — where to register
  `useSms` + `PreviewCard` (add in this stage with a temporary placeholder component)

## Implementation Details

### types.ts

```typescript
export interface SmsPayload {
  from: string
  to: string
  message: string
  gateway: string   // 'twilio' | 'vonage' | 'plivo' | 'generic'
}
```

### normalize-sms-event.ts

Follow `normalize-smtp-event.ts` exactly. Labels: `['sms']` prepended with time if available.

```typescript
export const normalizeSmsEvent = (event: ServerEvent<SmsPayload>): NormalizedEvent<SmsPayload> => {
  const normalizedEvent: NormalizedEvent<SmsPayload> = {
    id: event.uuid,
    type: EventTypes.Sms,
    labels: [EventTypes.Sms],
    origin: null,
    serverName: '',
    date: event.timestamp ? new Date(event.timestamp * 1000) : null,
    payload: event.payload,
  }
  if (normalizedEvent.date) {
    normalizedEvent.labels.unshift(moment(normalizedEvent.date).format('HH:mm:ss'))
  }
  return normalizedEvent
}
```

### mock payload (sms-twilio.json)

```json
{
  "uuid": "01234567-0000-0000-0000-000000000001",
  "type": "sms",
  "timestamp": 1700000000,
  "project": null,
  "payload": {
    "from": "+1234567890",
    "to": "+0987654321",
    "message": "Your verification code is 4892. Valid for 10 minutes.",
    "gateway": "twilio"
  }
}
```

### Registering in event mappers

In both `event-card-mapper.vue` and `event-page-mapper.vue`, add entries to
`EVENT_TYPE_COMPONENTS_MAP` following the exact same pattern as `EventTypes.Smtp`.
Use `PreviewCardDefault` as a temporary placeholder for the view component in Stage 4;
replace with the real component in Stage 5.

## Definition of Done

- [ ] `EventTypes.Sms = 'sms'` exists in `frontend/src/shared/types/events.ts`
- [ ] `SmsPayload` interface has `from`, `to`, `message`, `gateway`
- [ ] `normalizeSmsEvent` returns `NormalizedEvent<SmsPayload>` with correct `type` and `labels`
- [ ] `event-card-mapper.vue` has `[EventTypes.Sms]` entry (temporary or final view)
- [ ] `event-page-mapper.vue` has `[EventTypes.Sms]` entry (temporary or final view)
- [ ] All barrel exports compile without TypeScript errors
- [ ] Mock JSON is valid and matches `SmsPayload` shape

## Dependencies

**Requires**: Stage 3 (backend sends `type: 'sms'` events)
**Enables**: Stage 5 (UI components can import types and normalizer)
