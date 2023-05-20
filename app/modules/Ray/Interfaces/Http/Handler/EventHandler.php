<?php

declare(strict_types=1);

namespace Modules\Ray\Interfaces\Http\Handler;

use App\Application\Commands\ClearEvents;
use App\Application\Commands\HandleReceivedEvent;
use App\Application\Domain\ValueObjects\Uuid;
use App\Application\Service\HttpHandler\HandlerInterface;
use Carbon\CarbonInterval;
use Modules\Profiler\Application\EventHandlerInterface;
use Modules\Ray\Application\TypeEnum;
use Psr\Http\Message\ResponseInterface;
use Psr\Http\Message\ServerRequestInterface;
use Psr\SimpleCache\CacheInterface;
use Spiral\Cqrs\CommandBusInterface;
use Spiral\Http\ResponseWrapper;

final class EventHandler implements HandlerInterface
{
    public function __construct(
        private readonly ResponseWrapper $responseWrapper,
        private readonly EventHandlerInterface $handler,
        private readonly CommandBusInterface $commands,
        private readonly CacheInterface $cache,
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

        $uri = \ltrim($request->getUri()->getPath(), '/');

        return match (true) {
            $uri === '_availability_check' => $this->responseWrapper->create(400),
            \str_starts_with($uri, 'locks') => $this->handleLocks($request),
            default => $this->handleEvent($request),
        };
    }

    private function handleEvent(ServerRequestInterface $request): ResponseInterface
    {
        $event = \json_decode((string)$request->getBody(), true);

        $type = $event['payloads'][0]['type'] ?? null;

        if ($type === TypeEnum::CreateLock->value) {
            $hash = $event['payloads'][0]['content']['name'] ?? null;
            $this->cache->set($hash, 1, CarbonInterval::minute(5));
        } elseif ($type === TypeEnum::ClearAll->value) {
            $this->commands->dispatch(new ClearEvents(type: 'ray'));
        }

        $event = $this->handler->handle($event);

        $this->commands->dispatch(
            new HandleReceivedEvent(
                type: 'ray', payload: $event, uuid: Uuid::fromString($event['uuid'])
            )
        );

        return $this->responseWrapper->create(200);
    }

    private function handleLocks(ServerRequestInterface $request): ResponseInterface
    {
        $hash = \basename($request->getUri()->getPath());

        $lock = $this->cache->get($hash);

        if (!$lock || !\is_array($lock)) {
            return $this->responseWrapper->json(['active' => true, 'stop_execution' => false]);
        }

        return $this->responseWrapper->json($lock);
    }

    private function isValidRequest(ServerRequestInterface $request): bool
    {
        return $request->getHeaderLine('X-Buggregator-Event') === 'ray'
            || $request->getAttribute('event-type') === 'ray'
            || \str_starts_with($request->getUri()->getPath(), 'Ray')
            || $request->getUri()->getUserInfo() === 'ray';
    }
}
