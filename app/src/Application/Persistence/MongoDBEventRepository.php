<?php

declare(strict_types=1);

namespace App\Application\Persistence;

use App\Application\Domain\Entity\Json;
use App\Application\Domain\ValueObjects\Uuid;
use Carbon\Carbon;
use Modules\Events\Domain\Event;
use Modules\Events\Domain\EventRepositoryInterface;
use MongoDB\Collection;

final class MongoDBEventRepository implements EventRepositoryInterface
{
    public function __construct(
        private readonly Collection $collection,
    ) {}

    public function store(Event $event): bool
    {
        $result = $this->collection->insertOne([
            '_id' => (string)$event->getUuid(),
            'type' => $event->getType(),
            'project_id' => $event->getProjectId(),
            'date' => $event->getTimestamp(),
            'payload' => $event->getPayload()->jsonSerialize(),
        ]);

        return $result->getInsertedCount() > 0;
    }

    public function deleteAll(array $scope = []): void
    {
        $this->collection->deleteMany($scope);
    }

    public function deleteByPK(string $uuid): bool
    {
        $deleteResult = $this->collection->deleteOne(['_id' => $uuid]);

        return $deleteResult->getDeletedCount() > 0;
    }

    public function countAll(array $scope = []): int
    {
        return $this->collection->countDocuments($scope);
    }

    public function findAll(array $scope = [], array $orderBy = [], int $limit = 30, int $offset = 0): iterable
    {
        $cursor = $this->collection->find($scope, [
            'sort' => $this->mapOrderBy($orderBy),
            'limit' => $limit,
            'skip' => $offset,
        ]);

        foreach ($cursor as $document) {
            yield $this->mapDocumentInfoEvent($document);
        }
    }

    public function findByPK(mixed $uuid): ?Event
    {
        return $this->findOne(['_id' => $uuid]);
    }

    public function findOne(array $scope = []): ?Event
    {
        $document = $this->collection->findOne($scope);

        if ($document === null) {
            return null;
        }

        return $this->mapDocumentInfoEvent($document);
    }

    public function mapDocumentInfoEvent(\MongoDB\Model\BSONDocument $document): Event
    {
        return new Event(
            uuid: Uuid::fromString($document['_id']),
            type: $document['type'],
            payload: new Json((array)$document['payload']),
            timestamp: $document['date'],
            projectId: $document['project_id'],
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
