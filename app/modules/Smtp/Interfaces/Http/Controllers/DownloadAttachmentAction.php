<?php

declare(strict_types=1);

namespace Modules\Smtp\Interfaces\Http\Controllers;

use App\Application\Commands\FindEventByUuid;
use App\Application\Domain\ValueObjects\Uuid;
use App\Application\Exception\EntityNotFoundException;
use Modules\Events\Domain\Event;
use Nyholm\Psr7\Stream;
use Psr\Http\Message\ResponseInterface;
use Spiral\Cqrs\QueryBusInterface;
use Spiral\Http\Exception\ClientException\NotFoundException;
use Spiral\Http\ResponseWrapper;
use Spiral\Router\Annotation\Route;
use Spiral\Storage\StorageInterface;

final class DownloadAttachmentAction
{
    public function __construct(
        private readonly StorageInterface $storage,
    ) {
    }

    #[Route(route: 'smtp/<uuid>/attachment/<id>', name: 'smtp.attachment.download', group: 'api')]
    public function __invoke(
        QueryBusInterface $bus,
        ResponseWrapper $responseWrapper,
        Uuid $uuid,
        string $id,
    ): ResponseInterface {
        try {
            /** @var Event $event */
            $event = $bus->ask(new FindEventByUuid($uuid));
        } catch (EntityNotFoundException $e) {
            throw new NotFoundException($e->getMessage());
        }

        $attachment = $event->getPayload()->jsonSerialize()['attachments'][$id] ?? null;
        if ($attachment === null) {
            throw new NotFoundException('Attachment not found.');
        }

        $file = $this->storage->bucket('attachments')->file($attachment['uri']);
        $stream = $file->getStream();

        return $responseWrapper->create(200)
            ->withHeader('Content-Type', 'application/octet-stream')
            ->withHeader('Content-Length', $file->getSize())
            ->withHeader(
                'Content-Disposition',
                'attachment; filename="' . \addcslashes($attachment['name'], '"') . '"'
            )->withBody(Stream::create($stream));
    }
}
