<?php

declare(strict_types=1);

namespace Modules\Events\Interfaces\Http\Controllers;

use App\Application\Commands\FindEvents;
use App\Application\Event\EventTypeMapperInterface;
use App\Application\HTTP\Response\ErrorResource;
use Modules\Events\Interfaces\Http\Request\EventsRequest;
use Modules\Events\Interfaces\Http\Resources\EventPreviewCollection;
use Modules\Events\Interfaces\Http\Resources\EventPreviewResource;
use Spiral\Cqrs\QueryBusInterface;
use Spiral\Router\Annotation\Route;
use OpenApi\Attributes as OA;

#[OA\Get(
    path: '/api/events/preview',
    description: 'Retrieve all events preview',
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
            description: 'Filter by event type',
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
                            ref: EventPreviewResource::class,
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
final readonly class PreviewListAction
{
    #[Route(route: 'events/preview', name: 'events.preview.list', methods: 'GET', group: 'api')]
    public function __invoke(
        EventsRequest $request,
        QueryBusInterface $bus,
        EventTypeMapperInterface $mapper,
    ): EventPreviewCollection {
        return new EventPreviewCollection(
            $bus->ask(
                new FindEvents(type: $request->type, project: $request->project),
            ),
            $mapper,
        );
    }
}
