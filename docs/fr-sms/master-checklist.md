# Feature: SMS Gateway Interceptor

## Overview

Add an `Sms` module to Buggregator that acts as a fake SMS gateway. When developers test SMS
sending in their application (using any language / any SMS package), they point the webhook URL
to Buggregator instead of a real provider. Buggregator captures the request, normalizes it into
a simple `SmsMessage` (from, to, message, gateway name), stores it as a regular event, and
displays it in the UI.

No real SMS sending. No complex domain. Pure capture-and-display.

## Supported gateways (initial scope)

| Gateway | Detection | Key fields |
|---------|-----------|------------|
| Twilio | URL suffix `/sms/twilio` OR fields `MessageSid`+`Body` | `From`, `To`, `Body` |
| Vonage / Nexmo | URL suffix `/sms/vonage` OR fields `messageId`+`text` | `msisdn`→from, `to`, `text` |
| Plivo | URL suffix `/sms/plivo` OR fields `MessageUUID`+`Text` | `From`, `To`, `Text` |
| Generic | fallback — always matches | tries `from/to/message/body/text` |

## Stage Dependencies

```
Stage 1 (Domain DTO)
  → Stage 2 (Gateway Layer)
    → Stage 3 (HTTP Handler + Bootloader)
      → Stage 4 (Frontend Types + Normalizer)
        → Stage 5 (Frontend UI)
          → Stage 6 (Registration + Docs)
```

## Development Progress

---

### Stage 1: Domain DTO — SmsMessage

- [x] 1.1 Create `Modules\Sms\Domain\SmsMessage` readonly class implementing `JsonSerializable`
- [x] 1.2 Fields: `from: string`, `to: string`, `message: string`, `gateway: string`, `warnings: array`
- [x] 1.3 `jsonSerialize()` returns array with all four fields
- [x] 1.4 Verify no external dependencies (pure domain object)

**Notes**: _
**Status**: ✅ Complete
**Completed**: 2026-03-28

---

### Stage 2: Gateway Detection Layer

- [x] 2.1 Create `GatewayInterface` with `detect(body): bool` + `parse(body): SmsMessage` + `name(): string`
- [x] 2.2 Create `FieldMapGateway` — config-driven: detects by field presence, parses via field mapping
- [x] 2.3 Create `GatewayRegistry` — tries gateways in order, returns first match or null (→ pass through)
- [ ] 2.4 Unit-test each gateway with sample payloads

**Notes**: Replaced per-provider classes with single config-driven `FieldMapGateway`. Providers configured in bootloader.
**Status**: ✅ Core complete (tests pending)
**Completed**: 2026-03-28

---

### Stage 3: HTTP Handler + Bootloader

- [x] 3.1 Create `SmsHandler` implementing `HandlerInterface` (auto-discovered by tokenizer)
- [x] 3.2 Handler checks URL prefix `/sms`, extracts body (form or JSON), delegates to `GatewayRegistry`
- [x] 3.3 Dispatches `HandleReceivedEvent` with `type: 'sms'`, passes through if no gateway matched
- [x] 3.4 Create `EventTypeMapper` (`toPreview` returns from/to/message/gateway; `toSearchableText` from+to+message)
- [x] 3.5 Create `SmsBootloader` — binds `GatewayRegistry` with FieldMapGateway configs, registers `'sms'`

**Notes**: Handler passes through to HTTP dumps when: no body, no gateway match, or missing to/message.
**Status**: ✅ Complete
**Completed**: 2026-03-28

---

### Stage 4: Frontend — Types + Normalizer

- [x] 4.1 Add `Sms = 'sms'` to `EventTypes` enum in `frontend/src/shared/types/events.ts`
- [x] 4.2 Create `frontend/src/entities/sms/types.ts` — `SmsPayload` interface
- [x] 4.3 Create `normalize-sms-event.ts` following pattern of `normalize-smtp-event.ts`
- [x] 4.4 Create `frontend/src/entities/sms/lib/index.ts` + `index.ts` barrel exports
- [x] 4.5 Register `useSms` / `normalizeSmsEvent` in `event-card-mapper` and `event-page-mapper`

**Notes**: Placeholder components created for preview-card and sms-page (to be replaced in Stage 5). TS compiles clean.
**Status**: ✅ Complete
**Completed**: 2026-03-28

---

### Stage 5: Frontend — Preview Card + Detail Page

- [x] 5.1 Create `preview-card.vue` — shows `from → to` pill flow + gateway badge + message text preview
- [x] 5.2 Create `sms-page.vue` — full detail view (from, to, gateway badge, full message) with EventDetailLayout
- [x] 5.3 Barrel exports already in place from Stage 4
- [x] 5.4 Add Storybook stories for preview-card and sms-page (Twilio, Vonage, Generic variants)

**Notes**: Gateway badges colored per provider (twilio=red, vonage=blue, plivo=green, sinch=purple, generic=gray).
**Status**: ✅ Complete
**Completed**: 2026-03-28

---

### Stage 6: Registration + Documentation

- [x] 6.1 Add `sms` to `CLIENT_SUPPORTED_EVENTS` default in `AppBootloader` + `docker-compose.yaml`
- [x] 6.2 Add `SmsBootloader` to kernel bootloaders list
- [x] 6.3 Router already handles `/sms/:id` via generic `/:type/:id` route + EventTypes enum
- [x] 6.4 Integration doc at `docs/docs/config/sms.md` + added to VitePress sidebar
- [x] 6.5 SMS icon in sidebar (`sms.svg`)
- [x] 6.6 SMS in `PAGES_SETTINGS` + `initialCachedIds`
- [x] 6.7 Explicit provider URL routing (`/sms/{gateway}`) with validation + 422 response
- [x] 6.8 Validation warnings stored in payload and displayed in UI (preview-card + detail page)
- [x] 6.9 40+ provider field mappings in SmsBootloader
- [x] 6.10 28 example buttons in demo app (international + Russian + error examples)
- [x] 6.11 Event type color maps updated (SMS=fuchsia, Inspector=lime, VarDump=sky)

**Notes**: All stages complete. 40 PHP tests passing, 0 TS errors.
**Status**: ✅ Complete
**Completed**: 2026-03-28

---

## Codebase References

- `server/app/modules/Smtp/Application/Mapper/EventTypeMapper.php` — pattern for `toPreview` + `toSearchableText`
- `server/app/modules/Smtp/Application/SmtpBootloader.php` — bootloader pattern, `registry->register()`
- `server/app/modules/Sentry/Interfaces/Http/Handler/EventHandler.php` — HTTP handler that detects by header and dispatches
- `server/app/src/Application/Service/HttpHandler/HandlerInterface.php` — interface to implement
- `server/app/src/Application/Commands/HandleReceivedEvent.php` — command to dispatch
- `server/app/src/Application/Bootloader/AppBootloader.php` — `CLIENT_SUPPORTED_EVENTS` env default
- `frontend/src/shared/types/events.ts` — `EventTypes` enum to extend
- `frontend/src/entities/smtp/lib/normalize-smtp-event.ts` — normalizer pattern
- `frontend/src/entities/smtp/ui/preview-card/preview-card.vue` — preview card pattern
- `frontend/src/widgets/ui/event-card-mapper/event-card-mapper.vue` — where to register new entity
- `frontend/src/widgets/ui/event-page-mapper/event-page-mapper.vue` — where to register new page
