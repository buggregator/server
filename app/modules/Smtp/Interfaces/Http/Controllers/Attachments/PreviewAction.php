<?php

declare(strict_types=1);

namespace Modules\Smtp\Interfaces\Http\Controllers\Attachments;

use App\Application\Commands\FindEventByUuid;
use App\Application\Commands\FindSmtpAttachmentByUuid;
use App\Application\Domain\ValueObjects\Uuid;
use App\Application\HTTP\Response\ErrorResource;
use Modules\Smtp\Domain\AttachmentStorageInterface;
use Nyholm\Psr7\Stream;
use Psr\Http\Message\ResponseInterface;
use Spiral\Cqrs\QueryBusInterface;
use Spiral\Http\Exception\ClientException\ForbiddenException;
use Spiral\Http\ResponseWrapper;
use Spiral\Router\Annotation\Route;
use OpenApi\Attributes as OA;

#[OA\Get(
    path: '/api/smtp/{eventUuid}/attachments/preview/{uuid}',
    description: 'Preview an attachment by UUID',
    tags: ['Smtp'],
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
            ],
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
final readonly class PreviewAction
{
    public function __construct(
        private AttachmentStorageInterface $storage,
    ) {}

    #[Route(route: 'smtp/<eventUuid>/attachments/preview/<uuid>', name: 'smtp.attachments.preview', group: 'api_guest')]
    public function __invoke(
        QueryBusInterface $bus,
        ResponseWrapper $responseWrapper,
        Uuid $eventUuid,
        Uuid $uuid,
    ): ResponseInterface {
        $event = $bus->ask(new FindEventByUuid($eventUuid));
        $attachment = $bus->ask(new FindSmtpAttachmentByUuid($uuid));

        if (!$attachment->getEventUuid()->equals($event->getUuid())) {
            throw new ForbiddenException('Access denied.');
        }

        $stream = Stream::create($this->storage->getContent($attachment->getPath()));

        return $responseWrapper->create(200)
            ->withHeader('Content-Type', $attachment->getMime())
            ->withBody($stream);
    }
}
