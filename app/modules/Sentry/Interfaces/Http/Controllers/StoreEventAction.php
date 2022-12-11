<?php

declare(strict_types=1);

namespace Modules\Sentry\Interfaces\Http\Controllers;

use App\Application\Commands\HandleReceivedEvent;
use GuzzleHttp\Psr7\Stream;
use Http\Message\Encoding\GzipDecodeStream;
use Modules\Sentry\Application\EventHandlerInterface;
use Modules\Sentry\Application\GzippedStreamFactory;
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

    #[Route(route: '<projectId>/store', name: 'sentry.event.store', methods: ['POST'], group: 'api')]
    public function __invoke(
        ServerRequestInterface $request,
        CommandBusInterface $commands,
        QueryBusInterface $queryBus,
        int $projectId,
    ): void {
        $payload = $this->gzippedStreamFactory->createFromRequest($request)->getPayload();
        $event = $this->handler->handle($payload);
        $commands->dispatch(
            new HandleReceivedEvent(type: 'sentry', payload: $event)
        );
    }
}
