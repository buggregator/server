<?php

declare(strict_types=1);

namespace Modules\Profiler\Interfaces\Http\Handler;

use App\Application\Commands\HandleReceivedEvent;
use App\Application\Domain\ValueObjects\Uuid;
use App\Application\Event\EventType;
use App\Application\Service\HttpHandler\HandlerInterface;
use Modules\Profiler\Application\EventHandlerInterface;
use Modules\Profiler\Domain\ProfileManagerInterface;
use Psr\Http\Message\ResponseInterface;
use Psr\Http\Message\ServerRequestInterface;
use Spiral\Cqrs\CommandBusInterface;
use Spiral\Http\ResponseWrapper;

final readonly class EventHandler implements HandlerInterface
{
    public function __construct(
        private ResponseWrapper $responseWrapper,
        private EventHandlerInterface $handler,
        private CommandBusInterface $commands,
    ) {}

    public function priority(): int
    {
        return 0;
    }

    public function handle(ServerRequestInterface $request, \Closure $next): ResponseInterface
    {
        $eventType = $this->listenEvent($request);

        if ($eventType === null) {
            return $next($request);
        }

        $payload = \json_decode((string) $request->getBody(), true);
        $event = $this->handler->handle($payload);
        $profileUuid = Uuid::fromString($event['profile_uuid']);

        $this->commands->dispatch(
            new HandleReceivedEvent(
                type: $eventType->type,
                payload: $event,
                project: $eventType->project,
                uuid: $profileUuid,
            ),
        );

        return $this->responseWrapper->create(200);
    }

    private function listenEvent(ServerRequestInterface $request): ?EventType
    {
        /** @var EventType|null $event */
        $event = $request->getAttribute('event');

        if ($event?->type === 'profiler') {
            return $event;
        }

        if (
            $request->hasHeader('X-Profiler-Dump')
            || \str_ends_with($request->getUri()->getPath(), 'profiler/store')
        ) {
            return new EventType(type: 'profiler');
        }

        return null;
    }
}
