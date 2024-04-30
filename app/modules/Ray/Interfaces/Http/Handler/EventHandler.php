<?php

declare(strict_types=1);

namespace Modules\Ray\Interfaces\Http\Handler;

use App\Application\Commands\ClearEvents;
use App\Application\Commands\HandleReceivedEvent;
use App\Application\Domain\ValueObjects\Uuid;
use App\Application\Event\EventType;
use App\Application\Service\HttpHandler\HandlerInterface;
use Carbon\CarbonInterval;
use Modules\Ray\Application\EventHandlerInterface;
use Modules\Ray\Application\TypeEnum;
use Psr\Http\Message\ResponseInterface;
use Psr\Http\Message\ServerRequestInterface;
use Psr\SimpleCache\CacheInterface;
use Spiral\Cqrs\CommandBusInterface;
use Spiral\Http\ResponseWrapper;

final readonly class EventHandler implements HandlerInterface
{
    public function __construct(
        private ResponseWrapper $responseWrapper,
        private EventHandlerInterface $handler,
        private CommandBusInterface $commands,
        private CacheInterface $cache,
    ) {}

    public function priority(): int
    {
        return 0;
    }

    public function handle(ServerRequestInterface $request, \Closure $next): ResponseInterface
    {
        $event = $this->listenEvent($request);

        if ($event === null) {
            return $next($request);
        }

        $uri = \ltrim($request->getUri()->getPath(), '/');

        return match (true) {
            $uri === '_availability_check' => $this->responseWrapper->create(400),
            \str_starts_with($uri, 'locks') => $this->handleLocks($request),
            default => $this->handleEvent($request, $event),
        };
    }

    private function handleEvent(ServerRequestInterface $request, EventType $eventType): ResponseInterface
    {
        $event = \json_decode((string) $request->getBody(), true);

        $type = $event['payloads'][0]['type'] ?? null;

        if ($type === TypeEnum::CreateLock->value) {
            $hash = $event['payloads'][0]['content']['name'] ?? null;
            $this->cache->set($hash, 1, CarbonInterval::minute(5));
        } elseif ($type === TypeEnum::ClearAll->value) {
            $this->commands->dispatch(new ClearEvents(type: 'ray'));
            return $this->responseWrapper->create(200);
        }

        $event = $this->handler->handle($event);

        $this->commands->dispatch(
            new HandleReceivedEvent(
                type: $eventType->type,
                payload: $event,
                project: $eventType->project,
                uuid: Uuid::fromString($event['uuid']),
            ),
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

    private function listenEvent(ServerRequestInterface $request): ?EventType
    {
        /** @var EventType|null $event */
        $event = $request->getAttribute('event');

        if ($event?->type === 'ray') {
            return $event;
        }

        $userAgent = $request->getServerParams()['HTTP_USER_AGENT'] ?? '';

        if (\str_starts_with(\strtolower($userAgent), 'ray')) {
            return new EventType(type: 'ray');
        }

        return null;
    }
}
