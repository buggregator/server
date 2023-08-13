<?php

declare(strict_types=1);

namespace App\Application\Persistence;

use App\Application\Domain\ValueObjects\Uuid;
use Modules\Attachments\Domain\Attachment;
use Modules\Attachments\Domain\AttachmentRepositoryInterface;
use Spiral\Cache\CacheStorageProviderInterface;

final class CacheAttachmentRepository implements AttachmentRepositoryInterface
{
    private readonly CacheStorage $storage;

    public function __construct(
        CacheStorageProviderInterface $provider,
        $ttl = 60 * 60 * 2
    ) {
        $this->storage = new CacheStorage(
            $provider->storage('attachments'),
            'attachments',
            $ttl
        );
    }

    public function store(Attachment $event): bool
    {
        return $this->storage->store(
            $event->getUuid(),
            [],
            [
                'id' => (string)$event->getUuid(),
                'parent_id' => (string)$event->getParentUuid(),
                'path' => $event->getPath(),
                'size' => $event->getSize(),
                'mime_type' => $event->getMimeType(),
                'name' => $event->getFilename(),
            ]
        );
    }

    public function deleteAll(array $scope = []): void
    {
        $this->storage->deleteAll($scope);
    }

    public function deleteByPK(string $uuid): bool
    {
        return $this->storage->deleteByPK($uuid);
    }

    public function findByPK(mixed $id): ?object
    {
        $attachment = $this->storage->findByPK($uuid);

        if ($attachment === null) {
            return null;
        }

        return $this->mapDocumentInfoAttachment($attachment);
    }

    public function findOne(array $scope = []): ?object
    {
        $event = $this->storage->findOne($scope);

        if ($event === null) {
            return null;
        }

        return $this->mapDocumentInfoAttachment($event);
    }

    public function findAll(array $scope = []): iterable
    {
        $events = $this->storage->findAll($scope);

        foreach ($events as $document) {
            yield $this->mapDocumentInfoAttachment($document);
        }
    }

    private function mapDocumentInfoAttachment(array $document): Attachment
    {
        return new Attachment(
            uuid: Uuid::fromString($document['id']),
            parentUuid: Uuid::fromString($document['parent_id']),
            filename: $document['name'],
            path: $document['path'],
            mimeType: $document['mime_type'],
            size: $document['size'],
        );
    }
}
