<?php

declare(strict_types=1);

namespace Modules\Smtp\Application\Storage;

use Spiral\Storage\Exception\FileOperationException;
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

    public function store(Uuid $eventUuid, array $attachments): iterable
    {
        $result = [];
        foreach ($attachments as $attachment) {
            $file = $this->bucket->write(
                pathname: $eventUuid . '/' . $attachment->getFilename(),
                content: $attachment->getContent(),
            );

            $this->attachments->store(
                $a = $this->factory->create(
                    eventUuid: $eventUuid,
                    name: $attachment->getFilename(),
                    path: $file->getPathname(),
                    size: $file->getSize(),
                    mime: $file->getMimeType(),
                    id: $attachment->getContentId() ?? $attachment->getId(),
                ),
            );

            if ($attachment->getContentId() === null) {
                continue;
            }

            $result[$attachment->getContentId()] = \sprintf(
                '/api/smtp/%s/attachments/preview/%s',
                $eventUuid,
                $a->getUuid(),
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
