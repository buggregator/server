<?php

declare(strict_types=1);

namespace Modules\Sentry\Interfaces\Http\Handler;

use App\Application\Commands\HandleReceivedEvent;
use App\Application\Service\HttpHandler\HandlerInterface;
use Modules\Sentry\Application\EventHandlerInterface;
use Modules\Sentry\Application\PayloadParser;
use Psr\Http\Message\ResponseInterface;
use Psr\Http\Message\ServerRequestInterface;
use Spiral\Cqrs\CommandBusInterface;
use Spiral\Http\ResponseWrapper;

final class EventHandler implements HandlerInterface
{
    public function __construct(
        private readonly PayloadParser $payloadParser,
        private readonly ResponseWrapper $responseWrapper,
        private readonly EventHandlerInterface $handler,
        private readonly CommandBusInterface $commands,
    ) {
    }

    public function priority(): int
    {
        return 0;
    }

    public function handle(ServerRequestInterface $request, \Closure $next): ResponseInterface
    {
        if (!$this->isValidRequest($request)) {
            return $next($request);
        }

        $url = \rtrim($request->getUri()->getPath(), '/');

        $payloads = $this->payloadParser->parse($request);

        match (true) {
            \str_ends_with($url, '/envelope') => $this->handleEnvelope($payloads),
            \str_ends_with($url, '/store') => $this->handleEvent($payloads),
            default => null,
        };

        return $this->responseWrapper->create(200);
    }

    private function handleEvent(array $data): void
    {
        $event = $this->handler->handle($data[0]);

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
                'session' => null,
                'event' => $this->handleEvent([$data[2]]),
                default => null,
            };
        }
    }

    private function isValidRequest(ServerRequestInterface $request): bool
    {
        return $request->getHeaderLine('X-Buggregator-Event') === 'sentry'
            || $request->getAttribute('event-type') === 'sentry'
            || $request->hasHeader('X-Sentry-Auth')
            || $request->getUri()->getUserInfo() === 'sentry';
    }
}
