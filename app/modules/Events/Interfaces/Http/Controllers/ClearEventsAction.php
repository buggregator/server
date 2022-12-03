<?php

declare(strict_types=1);

namespace Modules\Events\Interfaces\Http\Controllers;

use App\Application\Commands\ClearEvents;
use Modules\Events\Interfaces\Http\Request\ClearEventsRequest;
use Spiral\Cqrs\CommandBusInterface;
use Spiral\Router\Annotation\Route;

final class ClearEventsAction
{
    #[Route(route: 'events', name: 'events.clear', methods: 'POST', group: 'api')]
    public function __invoke(ClearEventsRequest $request, CommandBusInterface $bus): void
    {
        $bus->dispatch(
            new ClearEvents(type: $request->type)
        );
    }
}
