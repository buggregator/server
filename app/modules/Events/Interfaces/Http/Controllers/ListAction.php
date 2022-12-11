<?php

declare(strict_types=1);

namespace Modules\Events\Interfaces\Http\Controllers;

use App\Application\Commands\FindEvents;
use Modules\Events\Interfaces\Http\Request\EventsRequest;
use Spiral\Cqrs\QueryBusInterface;
use Spiral\Router\Annotation\Route;

class ListAction
{
    #[Route(route: 'events', name: 'events.list', methods: 'GET', group: 'api')]
    public function __invoke(EventsRequest $request, QueryBusInterface $bus): EventCollection
    {
        return new EventCollection(
            $bus->ask(
                new FindEvents(type: $request->type)
            )
        );
    }
}
