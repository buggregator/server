<?php

declare(strict_types=1);

namespace Modules\Events\Interfaces\Http\Controllers;

use App\Application\Commands\FindEventsCursor;
use App\Application\HTTP\Response\ErrorResource;
use Modules\Events\Interfaces\Http\Request\EventsRequest;
use Modules\Events\Interfaces\Http\Resources\EventCursorCollection;
use Modules\Events\Interfaces\Http\Resources\EventResource;
use Modules\Events\Interfaces\Queries\EventsCursorResult;
use OpenApi\Attributes as OA;
use Spiral\Cqrs\QueryBusInterface;
use Spiral\Http\Request\InputManager;
use Spiral\Router\Annotation\Route;

#[OA\Get(
    path: '/api/events',
    description: 'Retrieve events with cursor pagination. Uses a composite cursor (timestamp + uuid) ordered by timestamp DESC, uuid DESC. Use meta.next_cursor as the cursor parameter to fetch the next page; meta.has_more indicates more data.',
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
        new OA\QueryParameter(
            name: 'limit',
            description: 'Page size (default 100, max 100)',
            required: false,
            schema: new OA\Schema(type: 'integer', minimum: 1, maximum: 100),
        ),
        new OA\QueryParameter(
            name: 'cursor',
            description: 'Opaque composite cursor (timestamp + uuid) from meta.next_cursor of the previous response',
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
                            ref: EventResource::class,
                        ),
                    ),
                    new OA\Property(
                        property: 'meta',
                        properties: [
                            new OA\Property(property: 'limit', type: 'integer'),
                            new OA\Property(property: 'has_more', type: 'boolean'),
                            new OA\Property(property: 'next_cursor', type: 'string', nullable: true),
                        ],
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
final readonly class ListAction
{
    #[Route(route: 'events', name: 'events.list', methods: 'GET', group: 'api')]
    public function __invoke(
        EventsRequest $request,
        InputManager $input,
        QueryBusInterface $bus,
    ): EventCursorCollection {
        $limit = $input->query->get('limit');
        $cursor = $input->query->get('cursor');

        /** @var EventsCursorResult $result */
        $result = $bus->ask(new FindEventsCursor(
            type: $request->type,
            project: $request->project,
            limit: $limit,
            cursor: $cursor,
        ));

        return new EventCursorCollection(
            $result->items,
            $result->limit,
            $result->hasMore,
            $result->nextCursor,
        );
    }
}
