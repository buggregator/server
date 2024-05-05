<?php

declare(strict_types=1);

namespace Modules\Smtp\Application\Storage;

use App\Application\Domain\ValueObjects\Uuid;
use Modules\Smtp\Domain\AttachmentFactoryInterface;
use Modules\Smtp\Domain\AttachmentRepositoryInterface;
use Modules\Smtp\Domain\AttachmentStorageInterface;
use Spiral\Storage\BucketInterface;

final readonly class AttachmentStorage implements AttachmentStorageInterface
{
    public function __construct(
        private BucketInterface $bucket,
        private AttachmentRepositoryInterface $attachments,
        private AttachmentFactoryInterface $factory,
    ) {}

    public function store(Uuid $eventUuid, array $attachments): void
    {
        foreach ($attachments as $attachment) {
            $file = $this->bucket->write(
                pathname: $eventUuid . '/' . $attachment->getFilename(),
                content: $attachment->getContent(),
            );

            $this->attachments->store(
                $this->factory->create(
                    eventUuid: $eventUuid,
                    name: $attachment->getFilename(),
                    path: $file->getPathname(),
                    size: $file->getSize(),
                    mime: $file->getMimeType(),
                    id: $attachment->getId(),
                ),
            );
        }
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
     * @throws \Spiral\Storage\Exception\FileOperationException
     */
    public function getContent(string $path)
    {
        return $this->bucket->getStream($path);
    }
}
