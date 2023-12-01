<?php

declare(strict_types=1);

namespace Modules\Events\Interfaces\Http\Controllers;

use App\Application\Commands\ClearEvents;
use App\Application\HTTP\Response\ResourceInterface;
use App\Application\HTTP\Response\SuccessResource;
use Modules\Events\Interfaces\Http\Request\ClearEventsRequest;
use Spiral\Cqrs\CommandBusInterface;
use Spiral\Router\Annotation\Route;

final class ClearAction
{
    #[Route(route: 'events', name: 'events.clear', methods: 'DELETE', group: 'api')]
    public function __invoke(ClearEventsRequest $request, CommandBusInterface $bus): ResourceInterface
    {
        $bus->dispatch(
            new ClearEvents(type: $request->type),
        );

        return new SuccessResource();
    }
}
