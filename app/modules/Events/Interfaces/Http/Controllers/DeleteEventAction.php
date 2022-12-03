<?php

declare(strict_types=1);

namespace Modules\Events\Interfaces\Http\Controllers;

use App\Application\Commands\DeleteEvent;
use App\Application\Domain\ValueObjects\Uuid;
use Spiral\Cqrs\CommandBusInterface;
use Spiral\Router\Annotation\Route;

final class DeleteEventAction
{
    #[Route(route: '/event/{uuid}', name: 'event.delete', group: 'api')]
    public function __invoke(CommandBusInterface $bus, Uuid $uuid): void
    {
        $bus->dispatch(new DeleteEvent($uuid));
    }
}
