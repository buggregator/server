<?php

declare(strict_types=1);

namespace Modules\Sentry\Interfaces\Http\Controllers;

use App\Application\Commands\HandleReceivedEvent;
use Http\Message\Encoding\GzipDecodeStream;
use Modules\Sentry\Application\EventHandlerInterface;
use Psr\Http\Message\ServerRequestInterface;
use Spiral\Cqrs\CommandBusInterface;
use Spiral\Cqrs\QueryBusInterface;
use Spiral\Router\Annotation\Route;

final class StoreEventAction
{
    #[Route(route: 'api/<projectId>/store', name: 'sentry.event.store', methods: ['POST'], group: 'api')]
    public function __invoke(
        ServerRequestInterface $request,
        CommandBusInterface $commands,
        QueryBusInterface $queryBus,
        EventHandlerInterface $handler,
        int $projectId,
    ): void {
        $stream = new GzipDecodeStream($request->getBody());
        $event = $handler->handle(\json_decode($stream->getContents(), true));
        $commands->dispatch(
            new HandleReceivedEvent(type: 'sentry', payload: $event)
        );
    }
}
