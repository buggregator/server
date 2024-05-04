<?php

declare(strict_types=1);

namespace Modules\Smtp\Interfaces\Http\Controllers\Attachments;

use App\Application\Commands\FindSmtpAttachmentsByEventUuid;
use App\Application\Domain\ValueObjects\Uuid;
use App\Application\HTTP\Response\ErrorResource;
use App\Application\HTTP\Response\ResourceCollection;
use App\Application\HTTP\Response\ResourceInterface;
use Modules\Smtp\Interfaces\Http\Resources\AttachmentResource;
use Spiral\Cqrs\QueryBusInterface;
use Spiral\Router\Annotation\Route;
use OpenApi\Attributes as OA;

#[OA\Get(
    path: '/api/smtp/{uuid}/attachments',
    description: 'Retrieve all attachments by event uuid',
    tags: ['Smtp'],
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
    #[Route(route: '/smtp/<uuid>/attachments', name: 'smtp.attachments.list', methods: ['GET'], group: 'api')]
    public function __invoke(QueryBusInterface $bus, Uuid $uuid): ResourceInterface
    {
        return new ResourceCollection(
            $bus->ask(new FindSmtpAttachmentsByEventUuid($uuid)),
            AttachmentResource::class,
        );
    }
}
