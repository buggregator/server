<?php

declare(strict_types=1);

namespace Modules\HttpDumps\Interfaces\Http\Controllers\Attachments;

use App\Application\Commands\FindHttpDumpAttachmentsByEventUuid;
use App\Application\Domain\ValueObjects\Uuid;
use App\Application\HTTP\Response\ErrorResource;
use App\Application\HTTP\Response\ResourceCollection;
use App\Application\HTTP\Response\ResourceInterface;
use Modules\HttpDumps\Interfaces\Http\Resources\AttachmentResource;
use Spiral\Cqrs\QueryBusInterface;
use Spiral\Router\Annotation\Route;
use OpenApi\Attributes as OA;

#[OA\Get(
    path: '/api/http-dumps/{uuid}/attachments',
    description: 'Retrieve all attachments by event uuid',
    tags: ['Http-Dumps'],
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
            content: new OA\JsonContent(
                properties: [
                    new OA\Property(
                        property: 'data',
                        type: 'array',
                        items: new OA\Items(
                            ref: AttachmentResource::class,
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
final readonly class ListAction
{
    #[Route(
        route: 'http-dumps/<uuid>/attachments',
        name: 'http-dumps.attachments.list',
        methods: ['GET'],
        group: 'api',
    )]
    public function __invoke(QueryBusInterface $bus, Uuid $uuid): ResourceInterface
    {
        return new ResourceCollection(
            $bus->ask(new FindHttpDumpAttachmentsByEventUuid($uuid)),
            AttachmentResource::class,
        );
    }
}
