<?php

declare(strict_types=1);

namespace Modules\HttpDumps\Application\Storage;

use Spiral\Storage\Exception\FileOperationException;
use App\Application\Domain\ValueObjects\Uuid;
use Modules\HttpDumps\Domain\AttachmentFactoryInterface;
use Modules\HttpDumps\Domain\AttachmentRepositoryInterface;
use Modules\HttpDumps\Domain\AttachmentStorageInterface;
use Spiral\Storage\BucketInterface;

final readonly class AttachmentStorage implements AttachmentStorageInterface
{
    public function __construct(
        private BucketInterface $bucket,
        private AttachmentRepositoryInterface $attachments,
        private AttachmentFactoryInterface $factory,
    ) {}

    public function store(Uuid $eventUuid, array $attachments): array
    {
        $result = [];

        foreach ($attachments as $attachment) {
            if ($attachment->getError() !== UPLOAD_ERR_OK) {
                // TODO: Log error
                continue;
            }

            $file = $this->bucket->write(
                pathname: $eventUuid . '/' . $attachment->getClientFilename(),
                content: $attachment->getStream(),
            );

            $this->attachments->store(
                $result[] = $this->factory->create(
                    eventUuid: $eventUuid,
                    name: $attachment->getClientFilename(),
                    path: $file->getPathname(),
                    size: $file->getSize(),
                    mime: $file->getMimeType(),
                ),
            );
        }

        return $result;
    }

    public function deleteByEvent(Uuid $eventUuid): void
    {
        $attachments = $this->attachments->findByEvent($eventUuid);
        foreach ($attachments as $attachment) {
            $this->bucket->delete(
                pathname: $attachment->getPath(),
                clean: true,
            );
        }
    }

    /**
     * @throws FileOperationException
     */
    public function getContent(string $path)
    {
        return $this->bucket->getStream($path);
    }
}
