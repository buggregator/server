<?php

declare(strict_types=1);

namespace Modules\Events\Interfaces\Http\Controllers;

use App\Application\Commands\CountEventsByType;
use App\Application\HTTP\Response\ErrorResource;
use Modules\Events\Interfaces\Http\Request\EventsRequest;
use Modules\Events\Interfaces\Http\Resources\EventTypeCountCollection;
use Modules\Events\Interfaces\Http\Resources\EventTypeCountResource;
use OpenApi\Attributes as OA;
use Spiral\Cqrs\QueryBusInterface;
use Spiral\Router\Annotation\Route;

#[OA\Get(
    path: '/api/events/type-counts',
    description: 'Retrieve event counts grouped by type',
    tags: ['Events'],
    parameters: [
        new OA\QueryParameter(
            name: 'type',
            description: 'Filter by event type',
            required: false,
            schema: new OA\Schema(type: 'string'),
        ),
        new OA\QueryParameter(
            name: 'project',
            description: 'Filter by event project',
            required: false,
            schema: new OA\Schema(type: 'string'),
        ),
    ],
    responses: [
        new OA\Response(
            response: 200,
            description: 'Success',
            content: new OA\JsonContent(
                properties: [
                    new OA\Property(
                        property: 'data',
                        type: 'array',
                        items: new OA\Items(
                            ref: EventTypeCountResource::class,
                        ),
                    ),
                    new OA\Property(
                        property: 'meta',
                        ref: '#/components/schemas/ResponseMeta',
                        type: 'object',
                    ),
                ],
            ),
        ),
        new OA\Response(
            response: 404,
            description: 'Not found',
            content: new OA\JsonContent(
                ref: ErrorResource::class,
            ),
        ),
    ],
)]
final readonly class TypeCountsAction
{
    #[Route(route: 'events/type-counts', name: 'events.type-counts', methods: 'GET', group: 'api')]
    public function __invoke(
        EventsRequest $request,
        QueryBusInterface $bus,
    ): EventTypeCountCollection {
        return new EventTypeCountCollection(
            $bus->ask(
                new CountEventsByType(type: $request->type, project: $request->project),
            ),
        );
    }
}
