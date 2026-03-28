# Stage 6: Registration + Documentation

## Overview

Final wiring: add `SmsBootloader` to the kernel, expose `sms` in the client supported events
default, add the Vue router entry, and write a one-page integration guide for users.

## Files

MODIFY:
- `server/app/app/src/Application/Kernel.php` — add `SmsBootloader` to bootloaders list
- `server/app/src/Application/Bootloader/AppBootloader.php` — add `sms` to `CLIENT_SUPPORTED_EVENTS`
- `frontend/src/app/router/` — add `/sms/:id` route pointing to the events page with SMS type

CREATE:
- `server/docs/fr-sms/integration-guide.md` — how to configure each supported gateway

## Code References

- `server/app/app/src/Application/Kernel.php` — find where other module bootloaders like
  `SmtpBootloader`, `SentryBootloader` are listed; add `SmsBootloader` in the same place
- `server/app/src/Application/Bootloader/AppBootloader.php` — `CLIENT_SUPPORTED_EVENTS` string,
  add `sms` to the comma-separated default value
- `frontend/src/app/router/` — find SMTP route as the pattern for adding `/sms/:id`

## Implementation Details

### Kernel.php change

Add `\Modules\Sms\Application\SmsBootloader::class` to the bootloaders array, grouped with
other module bootloaders (Smtp, Sentry, etc.).

### AppBootloader.php change

```php
// Before:
'CLIENT_SUPPORTED_EVENTS', 'http-dump,inspector,monolog,profiler,ray,sentry,smtp,var-dump'
// After:
'CLIENT_SUPPORTED_EVENTS', 'http-dump,inspector,monolog,profiler,ray,sentry,sms,smtp,var-dump'
```

Alphabetical order.

### Integration guide (key content)

```markdown
## SMS Gateway — Integration Guide

Point your SMS webhook URL to Buggregator instead of sending real SMS.

### Twilio
In the Twilio console, set the webhook URL for incoming messages to:
  http://your-buggregator:8000/sms/twilio
Or use the Twilio SDK and override the base URL:
  No SDK config needed — just set the webhook on the number.

### Vonage / Nexmo
  http://your-buggregator:8000/sms/vonage

### Plivo
  http://your-buggregator:8000/sms/plivo

### Generic (any provider)
POST to `http://your-buggregator:8000/sms` with any body containing:
  from, to, message (or body/text/Body/Text)
Both JSON and form-encoded are accepted.
```

## Definition of Done

- [ ] `SmsBootloader` is listed in `Kernel.php` bootloaders
- [ ] `sms` appears in `CLIENT_SUPPORTED_EVENTS` default value in `AppBootloader`
- [ ] Vue router has a `/sms/:id` route
- [ ] Sending a test request to `POST /sms/twilio` with Twilio-shaped body creates a visible
      event in the Buggregator UI
- [ ] `integration-guide.md` documents all four endpoint variants with example payloads

## Dependencies

**Requires**: Stage 3 (backend), Stage 5 (frontend UI complete)
**Enables**: Feature fully usable end-to-end
