# Stage 3: HTTP Handler + EventTypeMapper + Bootloader

## Overview

Wire everything together on the PHP side. `SmsHandler` is the PSR-7 middleware that intercepts
incoming requests to `/sms*` paths, uses `GatewayRegistry` to detect and parse the payload, and
dispatches `HandleReceivedEvent`. `EventTypeMapper` teaches Buggregator how to generate a preview
and searchable text from an SMS payload. `SmsBootloader` registers the gateway bindings and
registers the event type.

`SmsHandler` implements `HandlerInterface` and is **auto-discovered** by Spiral's tokenizer —
no manual registration needed in the pipeline.

## Files

CREATE:
- `server/app/modules/Sms/Interfaces/Http/Handler/SmsHandler.php`
- `server/app/modules/Sms/Application/Mapper/EventTypeMapper.php`
- `server/app/modules/Sms/Application/SmsBootloader.php`

## Code References

- `server/app/modules/Sentry/Interfaces/Http/Handler/EventHandler.php` — canonical HTTP handler
  pattern: check condition → call `$next($request)` if not matching → dispatch command → return 200
- `server/app/src/Application/Service/HttpHandler/HandlerInterface.php` — interface to implement;
  `priority()` determines order in the pipeline (use `0` like other handlers)
- `server/app/src/Application/Commands/HandleReceivedEvent.php` — command constructor signature
- `server/app/modules/Smtp/Application/Mapper/EventTypeMapper.php` — `toPreview` / `toSearchableText` pattern
- `server/app/modules/Smtp/Application/SmtpBootloader.php` — bootloader pattern for `defineSingletons` + `boot`

## Implementation Details

### SmsHandler

```php
#[Singleton]
final readonly class SmsHandler implements HandlerInterface
{
    public function __construct(
        private GatewayRegistry $registry,
        private CommandBusInterface $commands,
        private ResponseWrapper $responseWrapper,
    ) {}

    public function priority(): int { return 0; }

    public function handle(ServerRequestInterface $request, \Closure $next): ResponseInterface
    {
        $path = \rtrim($request->getUri()->getPath(), '/');

        // Only handle paths that start with /sms
        if (!\str_starts_with($path, '/sms')) {
            return $next($request);
        }

        // Extract suffix: /sms → '', /sms/twilio → 'twilio', /sms/vonage/project → 'vonage'
        $parts = \explode('/', \ltrim($path, '/'));
        $urlSuffix = $parts[1] ?? '';           // segment after 'sms'
        $project   = $parts[2] ?? null;         // optional project segment

        $gateway = $this->registry->detect($request, $urlSuffix);
        $sms     = $gateway->parse($request);

        $this->commands->dispatch(
            new HandleReceivedEvent(
                type: 'sms',
                payload: $sms,
                project: $project,
            ),
        );

        return $this->responseWrapper->create(200);
    }
}
```

### EventTypeMapper

```php
final readonly class EventTypeMapper implements EventTypeMapperInterface
{
    public function toPreview(string $type, array|\JsonSerializable $payload): array
    {
        $data = $payload instanceof \JsonSerializable ? $payload->jsonSerialize() : $payload;
        return [
            'from'    => $data['from'],
            'to'      => $data['to'],
            'message' => $data['message'],
            'gateway' => $data['gateway'],
        ];
    }

    public function toSearchableText(string $type, array|\JsonSerializable $payload): string
    {
        $data = $payload instanceof \JsonSerializable ? $payload->jsonSerialize() : $payload;
        return \implode(' ', \array_filter([
            $data['from']    ?? null,
            $data['to']      ?? null,
            $data['message'] ?? null,
        ]));
    }
}
```

### SmsBootloader

```php
final class SmsBootloader extends Bootloader
{
    public function defineSingletons(): array
    {
        return [
            GatewayRegistry::class => static fn(): GatewayRegistry => new GatewayRegistry([
                new TwilioGateway(),
                new VonageGateway(),
                new PlivoGateway(),
                new GenericGateway(),
            ]),
        ];
    }

    public function boot(EventTypeRegistryInterface $registry): void
    {
        $registry->register('sms', new EventTypeMapper());
    }
}
```

## Definition of Done

- [ ] `SmsHandler` implements `HandlerInterface`, has `#[Singleton]` attribute
- [ ] Requests to `/sms`, `/sms/twilio`, `/sms/vonage`, `/sms/plivo` are intercepted
- [ ] Requests to other paths pass through via `$next($request)`
- [ ] Project extracted from third path segment when present
- [ ] `HandleReceivedEvent` dispatched with `type: 'sms'`
- [ ] `EventTypeMapper::toPreview()` returns `from`, `to`, `message`, `gateway`
- [ ] `EventTypeMapper::toSearchableText()` concatenates `from`, `to`, `message`
- [ ] `SmsBootloader` registers all four gateways in correct order
- [ ] `SmsBootloader` calls `$registry->register('sms', ...)`
- [ ] Server returns HTTP 200 for captured SMS requests

## Dependencies

**Requires**: Stage 1 (`SmsMessage`), Stage 2 (`GatewayRegistry` + gateways)
**Enables**: Stage 4 (frontend can receive `sms` events), Stage 6 (bootloader added to kernel)
