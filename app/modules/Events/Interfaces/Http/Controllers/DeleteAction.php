<?php

declare(strict_types=1);

namespace Modules\Events\Interfaces\Http\Controllers;

use App\Application\Commands\DeleteEvent;
use App\Application\Domain\ValueObjects\Uuid;
use App\Application\HTTP\Response\ResourceInterface;
use App\Application\HTTP\Response\SuccessResource;
use Spiral\Cqrs\CommandBusInterface;
use Spiral\Router\Annotation\Route;
use OpenApi\Attributes as OA;

#[OA\Delete(
    path: '/api/event/{uuid}',
    description: 'Delete an event by UUID',
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
            content: new OA\JsonContent(ref: SuccessResource::class),
        ),
    ],
)]
final class DeleteAction
{
    #[Route(route: 'event/<uuid>', name: 'event.delete', methods: 'DELETE', group: 'api')]
    public function __invoke(CommandBusInterface $bus, Uuid $uuid): ResourceInterface
    {
        $bus->dispatch(
            new DeleteEvent($uuid),
        );

        return new SuccessResource();
    }
}
