<?php

declare(strict_types=1);

namespace Modules\HttpDumps\Interfaces\Http\Controllers\Attachments;

use App\Application\Commands\FindEventByUuid;
use App\Application\Commands\FindHttpDumpAttachmentByUuid;
use App\Application\Domain\ValueObjects\Uuid;
use App\Application\HTTP\Response\ErrorResource;
use Modules\HttpDumps\Domain\AttachmentStorageInterface;
use Nyholm\Psr7\Stream;
use Psr\Http\Message\ResponseInterface;
use Spiral\Cqrs\QueryBusInterface;
use Spiral\Http\Exception\ClientException\ForbiddenException;
use Spiral\Http\ResponseWrapper;
use Spiral\Router\Annotation\Route;
use OpenApi\Attributes as OA;

#[OA\Get(
    path: '/api/http-dumps/{eventUuid}/attachments/{uuid}',
    description: 'Download an attachment by UUID',
    tags: ['Http-Dumps'],
    parameters: [
        new OA\PathParameter(
            name: 'eventUuid',
            description: 'Event UUID',
            required: true,
            schema: new OA\Schema(type: 'string', format: 'uuid'),
        ),
        new OA\PathParameter(
            name: 'uuid',
            description: 'Attachment UUID',
            required: true,
            schema: new OA\Schema(type: 'string', format: 'uuid'),
        ),
    ],
    responses: [
        new OA\Response(
            response: 200,
            description: 'Success',
            headers: [
                new OA\Header(header: 'Content-Type', schema: new OA\Schema(type: 'string', format: 'binary')),
                new OA\Header(header: 'Content-Length', schema: new OA\Schema(type: 'string', format: 'binary')),
                new OA\Header(header: 'Content-Disposition', schema: new OA\Schema(type: 'string', format: 'binary')),
            ],
            content: new OA\MediaType(
                mediaType: 'application/octet-stream',
                schema: new OA\Schema(type: 'string', format: 'binary'),
            ),
        ),
        new OA\Response(
            response: 404,
            description: 'Not found',
            content: new OA\JsonContent(
                ref: ErrorResource::class,
            ),
        ),
        new OA\Response(
            response: 403,
            description: 'Access denied.',
            content: new OA\JsonContent(
                ref: ErrorResource::class,
            ),
        ),
    ],
)]
final readonly class DownloadAction
{
    public function __construct(
        private AttachmentStorageInterface $storage,
    ) {}

    #[Route(
        route: 'http-dumps/<eventUuid>/attachments/<uuid>',
        name: 'http-dumps.attachments.download',
        group: 'api_guest',
    )]
    public function __invoke(
        QueryBusInterface $bus,
        ResponseWrapper $responseWrapper,
        Uuid $eventUuid,
        Uuid $uuid,
    ): ResponseInterface {
        $event = $bus->ask(new FindEventByUuid($eventUuid));
        $attachment = $bus->ask(new FindHttpDumpAttachmentByUuid($uuid));

        if (!$attachment->getEventUuid()->equals($event->getUuid())) {
            throw new ForbiddenException('Access denied.');
        }

        $filename = \addcslashes($attachment->getFilename(), '"');
        $stream = Stream::create($this->storage->getContent($attachment->getPath()));

        return $responseWrapper->create(200)
            ->withHeader('Content-Type', 'application/octet-stream')
            ->withHeader('Content-Length', (string) $stream->getSize())
            ->withHeader('Content-Disposition', \sprintf('attachment; filename="%s"', $filename))
            ->withBody($stream);
    }
}
