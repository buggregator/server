# Stage 2: Gateway Detection Layer

## Overview

Build the gateway detection and parsing layer. Each SMS provider sends differently-structured
HTTP requests. This stage introduces a small strategy pattern: `GatewayInterface` defines how
to detect and parse a provider's request; concrete implementations handle Twilio, Vonage, Plivo,
and a Generic fallback. `GatewayRegistry` chains them and returns the first match.

Detection is two-level: URL suffix first (explicit, set by user), then payload structure (auto).

## Files

CREATE:
- `server/app/modules/Sms/Application/Gateway/GatewayInterface.php` — contract
- `server/app/modules/Sms/Application/Gateway/GatewayRegistry.php` — ordered chain
- `server/app/modules/Sms/Application/Gateway/TwilioGateway.php`
- `server/app/modules/Sms/Application/Gateway/VonageGateway.php`
- `server/app/modules/Sms/Application/Gateway/PlivoGateway.php`
- `server/app/modules/Sms/Application/Gateway/GenericGateway.php` — always-match fallback

## Code References

- `server/app/modules/Sentry/Application/PayloadParser.php` — how to safely extract data from
  a PSR-7 request body (use `getParsedBody()` for form-encoded, `json_decode` for JSON)
- `server/app/modules/Smtp/Application/Mail/Message.php` — `SmsMessage` (Stage 1) is the return
  type for `parse()`, same readonly pattern

## Implementation Details

### GatewayInterface

```php
interface GatewayInterface
{
    // $urlSuffix = last path segment after /sms, e.g. 'twilio', 'vonage', or '' for generic
    public function detect(ServerRequestInterface $request, string $urlSuffix): bool;
    public function parse(ServerRequestInterface $request): SmsMessage;
    public function name(): string;
}
```

### Detection logic per gateway

| Gateway | URL match | Payload auto-detect |
|---------|-----------|---------------------|
| Twilio | `$urlSuffix === 'twilio'` | `isset($body['MessageSid'], $body['Body'])` |
| Vonage | `$urlSuffix === 'vonage'` | `isset($body['messageId'], $body['text'])` |
| Plivo | `$urlSuffix === 'plivo'` | `isset($body['MessageUUID'], $body['Text'])` |
| Generic | always true | — |

### Payload parsing

Twilio sends `application/x-www-form-urlencoded` → use `$request->getParsedBody()`.
Vonage sends JSON → `json_decode((string)$request->getBody(), true)`.
Plivo sends JSON → same.
Generic → try `getParsedBody()` first, fallback to JSON decode; extract first non-empty value
from keys: `from`, `From`, `sender`, `msisdn`; `to`, `To`, `recipient`; `body`, `Body`,
`text`, `Text`, `message`, `Message`.

### GatewayRegistry

```php
final readonly class GatewayRegistry
{
    /** @param GatewayInterface[] $gateways */
    public function __construct(private array $gateways) {}

    public function detect(ServerRequestInterface $request, string $urlSuffix): GatewayInterface
    {
        foreach ($this->gateways as $gateway) {
            if ($gateway->detect($request, $urlSuffix)) {
                return $gateway;
            }
        }
        // GenericGateway is always last and always matches — this is unreachable
        throw new \RuntimeException('No SMS gateway matched');
    }
}
```

Register gateways in order: Twilio → Vonage → Plivo → Generic.

## Definition of Done

- [ ] `GatewayInterface` defines `detect()`, `parse()`, `name()`
- [ ] `TwilioGateway`, `VonageGateway`, `PlivoGateway` detect by both URL suffix and payload fields
- [ ] `GenericGateway::detect()` always returns `true`
- [ ] `GatewayRegistry` iterates in order and returns the first matching gateway
- [ ] All `parse()` methods return a valid `SmsMessage` with non-empty strings
- [ ] `declare(strict_types=1)` and `final readonly` on all concrete classes

## Dependencies

**Requires**: Stage 1 (`SmsMessage` DTO)
**Enables**: Stage 3 (handler uses `GatewayRegistry`)
