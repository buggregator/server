<?php

declare(strict_types=1);

namespace Modules\Attachments\Interfaces\Http\Controllers;

use App\Application\Commands\FindAttachmentByUuid;
use App\Application\Domain\ValueObjects\Uuid;
use App\Application\Exception\EntityNotFoundException;
use Modules\Attachments\Domain\Attachment;
use Nyholm\Psr7\Stream;
use Psr\Http\Message\ResponseInterface;
use Spiral\Cqrs\QueryBusInterface;
use Spiral\Http\Exception\ClientException\NotFoundException;
use Spiral\Http\ResponseWrapper;
use Spiral\Router\Annotation\Route;
use Spiral\Storage\BucketInterface;
use Spiral\Storage\Exception\FileOperationException;
use Spiral\Storage\StorageInterface;

final class DownloadAction
{
    private readonly BucketInterface $bucket;

    public function __construct(
        StorageInterface $storage,
    ) {
        $this->bucket = $storage->bucket('attachments');
    }

    /**
     * @throws FileOperationException
     */
    #[Route(route: 'attachment/<uuid>/download', name: 'attachment.download', group: 'api')]
    public function __invoke(
        QueryBusInterface $bus,
        ResponseWrapper $responseWrapper,
        Uuid $uuid,
    ): ResponseInterface {
        try {
            /** @var Attachment $attachment */
            $attachment = $bus->ask(new FindAttachmentByUuid($uuid));
        } catch (EntityNotFoundException $e) {
            throw new NotFoundException($e->getMessage());
        }

        $file = $this->bucket->file($attachment->getPath());
        $stream = $file->getStream();

        return $responseWrapper->create(200)
            ->withHeader('Content-Type', 'application/octet-stream')
            ->withHeader('Content-Length', $file->getSize())
            ->withHeader(
                'Content-Disposition',
                'attachment; filename="' . \addcslashes($attachment->getFilename(), '"') . '"'
            )->withBody(Stream::create($stream));
    }
}
