<?php

declare(strict_types=1);

namespace Modules\Profiler\Interfaces\Http\Controllers;

use App\Application\Commands\HandleReceivedEvent;
use App\Application\HTTP\GzippedStreamFactory;
use Modules\Profiler\Application\EventHandlerInterface;
use Psr\Http\Message\ServerRequestInterface;
use Spiral\Cqrs\CommandBusInterface;
use Spiral\Cqrs\QueryBusInterface;
use Spiral\Router\Annotation\Route;

final class StoreEventAction
{
    public function __construct(
        private readonly GzippedStreamFactory $gzippedStreamFactory,
        private readonly EventHandlerInterface $handler
    ) {
    }

    #[Route(route: 'profiler/store', name: 'profiler.event.store', methods: ['POST'], group: 'api', priority: 80)]
    public function __invoke(
        ServerRequestInterface $request,
        CommandBusInterface $commands,
        QueryBusInterface $queryBus
    ): void {
        //$payload = $this->gzippedStreamFactory->createFromRequest($request)->getPayload();
        $payload = \json_decode((string) $request->getBody(), true);

        $event = $this->handler->handle($payload);
        $commands->dispatch(
            new HandleReceivedEvent(type: 'profiler', payload: $event)
        );
    }
}
