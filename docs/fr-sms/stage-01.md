# Stage 1: Domain DTO — SmsMessage

## Overview

Create the single domain object for the SMS module: `SmsMessage`. This is a pure,
framework-agnostic readonly DTO that represents a captured SMS event. No database entities,
no repositories, no external dependencies — just a value object that carries the four fields
we care about: who sent it, who received it, what the text was, and which gateway sent it.

## Files

CREATE:
- `server/app/modules/Sms/Domain/SmsMessage.php` — readonly DTO implementing `JsonSerializable`

## Code References

- `server/app/modules/Smtp/Application/Mail/Message.php` — same pattern: readonly class,
  constructor property promotion, `jsonSerialize()` returning a flat array. Follow this exactly
  but strip it down — no attachments, no raw body, no complex logic.

## Implementation Details

```php
final readonly class SmsMessage implements \JsonSerializable
{
    public function __construct(
        public string $from,
        public string $to,
        public string $message,
        public string $gateway,  // 'twilio' | 'vonage' | 'plivo' | 'generic'
    ) {}

    public function jsonSerialize(): array
    {
        return [
            'from'    => $this->from,
            'to'      => $this->to,
            'message' => $this->message,
            'gateway' => $this->gateway,
        ];
    }
}
```

Keep `gateway` as a plain string — no enum needed at this stage (YAGNI).

## Definition of Done

- [ ] `SmsMessage` exists at `Modules\Sms\Domain\SmsMessage`
- [ ] Class is `final readonly`, uses constructor property promotion
- [ ] `jsonSerialize()` returns exactly `from`, `to`, `message`, `gateway`
- [ ] Zero dependencies on any framework, service, or other module
- [ ] File has `declare(strict_types=1)`

## Dependencies

**Requires**: nothing
**Enables**: Stage 2 (gateways return `SmsMessage`), Stage 3 (handler uses it)
