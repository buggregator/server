<?php

declare(strict_types=1);

namespace Modules\Ray\Interfaces\Http\Controllers;

use App\Application\Commands\ClearEvents;
use App\Application\Commands\HandleReceivedEvent;
use App\Application\Domain\ValueObjects\Uuid;
use Carbon\CarbonInterval;
use Modules\Ray\Application\EventHandlerInterface;
use Modules\Ray\Application\TypeEnum;
use Psr\Http\Message\ServerRequestInterface;
use Psr\SimpleCache\CacheInterface;
use Spiral\Cqrs\CommandBusInterface;
use Spiral\Cqrs\QueryBusInterface;
use Spiral\Http\Request\InputManager;
use Spiral\Router\Annotation\Route;

final class StoreEventAction
{
    public function __construct(
        private readonly EventHandlerInterface $handler
    ) {
    }

    #[Route(route: '/', name: 'ray.event.store', methods: 'POST')]
    public function __invoke(
        InputManager $request,
        CommandBusInterface $commands,
        CacheInterface $cache,
        EventHandlerInterface $handler,
        QueryBusInterface $queryBus,
        ServerRequestInterface $r
    ): void {
        $event = \json_decode((string)$r->getBody(), true);

        $type = $event['payloads'][0]['type'] ?? null;

        if ($type === TypeEnum::CreateLock->value) {
            $hash = $event['payloads'][0]['content']['name'] ?? null;
            $cache->set($hash, 1, CarbonInterval::minute(5));
            return;
        } elseif ($type === TypeEnum::ClearAll->value) {
            $commands->dispatch(new ClearEvents(type: 'ray'));
            return;
        }

        $event = $this->handler->handle($event);

        $commands->dispatch(
            new HandleReceivedEvent(
                type: 'ray', payload: $event, uuid: Uuid::fromString($event['uuid'])
            )
        );
    }
}
