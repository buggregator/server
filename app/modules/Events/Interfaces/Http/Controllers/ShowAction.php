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
use OpenApi\Attributes as OA;

#[OA\Get(
    path: '/api/event/{uuid}',
    description: 'Retrieve an event by UUID',
    tags: ['Events'],
    parameters: [
        new OA\PathParameter(
            name: 'uuid',
            description: 'Event UUID',
            required: true,
            schema: new OA\Schema(type: 'string', format: 'uuid'),
        ),
    ],
    responses: [
        new OA\Response(
            response: 200,
            description: 'Success',
            content: new OA\JsonContent(ref: EventResource::class),
        ),
        new OA\Response(
            response: 404,
            description: 'Not found',
            content: new OA\JsonContent(
                ref: '#/components/schemas/NotFoundError',
            ),
        ),
    ]
)]
final class ShowAction
{
    #[Route(route: 'event/<uuid>', name: 'event.show', methods: 'GET', group: 'api')]
    public function __invoke(QueryBusInterface $bus, Uuid $uuid): EventResource
    {
        try {
            return new EventResource(
                $bus->ask(new FindEventByUuid($uuid)),
            );
        } catch (EntityNotFoundException $e) {
            throw new NotFoundException($e->getMessage());
        }
    }
}
