<?php

declare(strict_types=1);

namespace Modules\Events\Interfaces\Http\Controllers;

use App\Application\Commands\FindEventByUuid;
use App\Application\Domain\ValueObjects\Uuid;
use App\Application\Exception\EntityNotFoundException;
use Modules\Events\Interfaces\Http\Resources\EventResource;
use Spiral\Cqrs\QueryBusInterface;
use Spiral\Http\Exception\ClientException\NotFoundException;
use Spiral\Router\Annotation\Route;

final class ShowAction
{
    #[Route(route: 'event/<uuid>', name: 'event.show', methods: 'GET', group: 'api')]
    public function __invoke(QueryBusInterface $bus, Uuid $uuid): EventResource
    {
        try {
            return new EventResource(
                $bus->ask(new FindEventByUuid($uuid))
            );
        } catch (EntityNotFoundException $e) {
            throw new NotFoundException($e->getMessage());
        }
    }
}
