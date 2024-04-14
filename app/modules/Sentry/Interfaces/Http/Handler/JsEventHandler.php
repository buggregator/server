<?php

declare(strict_types=1);

namespace Modules\Sentry\Interfaces\Http\Handler;

use App\Application\Commands\HandleReceivedEvent;
use App\Application\Service\HttpHandler\HandlerInterface;
use Modules\Sentry\Application\EventHandlerInterface;
use Psr\Http\Message\ResponseInterface;
use Psr\Http\Message\ServerRequestInterface;
use Spiral\Cqrs\CommandBusInterface;
use Spiral\Http\ResponseWrapper;

final readonly class JsEventHandler implements HandlerInterface
{
    public function __construct(
        private ResponseWrapper $responseWrapper,
        private EventHandlerInterface $handler,
        private CommandBusInterface $commands,
    ) {
    }

    public function priority(): int
    {
        return 1;
    }

    public function handle(ServerRequestInterface $request, \Closure $next): ResponseInterface
    {
        if (!$this->isValidRequest($request)) {
            return $next($request);
        }

        $payloads = \array_map(
            static fn(string $payload): array => \json_decode($payload, true, 512, \JSON_THROW_ON_ERROR),
            \explode("\n", (string)$request->getBody()),
        );

        if (!isset($payloads[1]['type'])) {
            return $this->responseWrapper->create(404);
        }

        match ($payloads[1]['type']) {
            'event' => $this->handleEvent($payloads),
            'transaction' => $this->handleEnvelope($payloads),
            default => null,
        };

        return $this->responseWrapper->create(200);
    }

    private function handleEvent(array $data): void
    {
        $event = $this->handler->handle($data[2]);

        $this->commands->dispatch(
            new HandleReceivedEvent(type: 'sentry', payload: $event),
        );
    }

    /**
     * TODO handle sentry transaction and session
     */
    private function handleEnvelope(array $data): void
    {
        if (\count($data) == 3) {
            match ($data[1]['type']) {
                'transaction' => null,
            };
        }
    }

    private function isValidRequest(ServerRequestInterface $request): bool
    {
        return isset($request->getQueryParams()['sentry_key']);
    }
}
