<?php

declare(strict_types=1);

namespace Modules\Events\Interfaces\Http\Controllers;

use App\Application\Commands\ClearEvents;
use App\Application\HTTP\Response\ResourceInterface;
use App\Application\HTTP\Response\SuccessResource;
use Modules\Events\Interfaces\Http\Request\ClearEventsRequest;
use Spiral\Cqrs\CommandBusInterface;
use Spiral\Router\Annotation\Route;
use OpenApi\Attributes as OA;

#[OA\Delete(
    path: '/api/events',
    description: 'Clear all events',
    requestBody: new OA\RequestBody(
        content: new OA\JsonContent(ref: ClearEventsRequest::class),
    ),
    tags: ['Events'],
    responses: [
        new OA\Response(
            response: 200,
            description: 'Success',
            content: new OA\JsonContent(ref: SuccessResource::class),
        ),
    ]
)]
final readonly class ClearAction
{
    #[Route(route: 'events', name: 'events.clear', methods: 'DELETE', group: 'api')]
    public function __invoke(ClearEventsRequest $request, CommandBusInterface $bus): ResourceInterface
    {
        $bus->dispatch(
            new ClearEvents(
                type: $request->type,
                project: $request->project,
                uuids: $request->uuids,
            ),
        );

        return new SuccessResource();
    }
}
