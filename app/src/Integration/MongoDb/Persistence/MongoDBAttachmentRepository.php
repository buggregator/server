<?php

declare(strict_types=1);

namespace App\Integration\MongoDb\Persistence;

use App\Application\Domain\ValueObjects\Uuid;
use Modules\Smtp\Domain\Attachment;
use Modules\Smtp\Domain\AttachmentRepositoryInterface;
use MongoDB\Collection;
use MongoDB\Model\BSONDocument;

/**
 * todo: cover with tests
 */
final readonly class MongoDBAttachmentRepository implements AttachmentRepositoryInterface
{
    public function __construct(
        private Collection $collection,
    ) {}

    public function store(Attachment $attachment): bool
    {
        $result = $this->collection->insertOne([
            '_id' => (string) $attachment->getUuid(),
            'event_uuid' => (string) $attachment->getEventUuid(),
            'name' => $attachment->getName(),
            'path' => $attachment->getPath(),
            'size' => $attachment->getSize(),
            'mime' => $attachment->getMime(),
            'id' => $attachment->getId(),
        ]);

        return $result->getInsertedCount() > 0;
    }

    public function findByEvent(Uuid $uuid): iterable
    {
        $cursor = $this->collection->find([
            'event_uuid' => (string) $uuid,
        ]);

        foreach ($cursor as $document) {
            yield $this->mapDocumentIntoAttachment($document);
        }
    }

    public function deleteByEvent(Uuid $uuid): void
    {
        $this->collection->deleteMany(['event_uuid' => (string) $uuid]);
    }

    public function findByPK(mixed $id): ?object
    {
        return $this->findOne(['_id' => $id]);
    }

    /**
     * @return Attachment|null
     */
    public function findOne(array $scope = []): ?object
    {
        /** @var BSONDocument|null $document */
        $document = $this->collection->findOne($scope);

        if ($document === null) {
            return null;
        }

        return $this->mapDocumentIntoAttachment($document);
    }

    public function findAll(array $scope = [], array $orderBy = [], int $limit = 30, int $offset = 0): iterable
    {
        $cursor = $this->collection->find($scope, [
            'sort' => $this->mapOrderBy($orderBy),
            'limit' => $limit,
            'skip' => $offset,
        ]);

        foreach ($cursor as $document) {
            yield $this->mapDocumentIntoAttachment($document);
        }
    }

    private function mapDocumentIntoAttachment(BSONDocument $document): Attachment
    {
        /** @psalm-suppress InternalMethod */
        return new Attachment(
            uuid: new Uuid($document['_id']),
            eventUuid: new Uuid($document['event_uuid']),
            name: $document['name'],
            path: $document['path'],
            size: (int) $document['size'],
            mime: $document['mime'],
            id: $document['id'],
        );
    }

    private function mapOrderBy(array $orderBy): array
    {
        $result = [];

        foreach ($orderBy as $key => $order) {
            $result[$key] = $order === 'asc' ? 1 : -1;
        }

        return $result;
    }

}
