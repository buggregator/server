<?php

declare(strict_types=1);

namespace Modules\Profiler\Interfaces\Http\Handler;

use App\Application\Commands\HandleReceivedEvent;
use App\Application\Service\HttpHandler\HandlerInterface;
use Modules\Profiler\Application\EventHandlerInterface;
use Psr\Http\Message\ResponseInterface;
use Psr\Http\Message\ServerRequestInterface;
use Spiral\Cqrs\CommandBusInterface;
use Spiral\Http\ResponseWrapper;

final class EventHandler implements HandlerInterface
{
    public function __construct(
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

        $payload = \json_decode((string)$request->getBody(), true);
        $event = $this->handler->handle($payload);

        $this->commands->dispatch(
            new HandleReceivedEvent(type: 'profiler', payload: $event)
        );

        return $this->responseWrapper->create(200);
    }

    private function isValidRequest(ServerRequestInterface $request): bool
    {
        return $request->getHeaderLine('X-Buggregator-Event') === 'profiler'
            || $request->getAttribute('event-type') === 'profiler'
            || $request->getUri()->getUserInfo() === 'profiler'
            || $request->hasHeader('X-Profiler-Dump')
            || \str_ends_with($request->getUri()->getPath(), 'profiler/store');
    }
}
