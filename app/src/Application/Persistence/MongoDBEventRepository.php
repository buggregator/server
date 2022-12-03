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
    ) {
    }

    public function store(Event $event): bool
    {
        $result = $this->collection->insertOne([
            '_id' => (string)$event->getUuid(),
            'type' => $event->getType(),
            'project_id' => $event->getProjectId(),
            'created_at' => $event->getDate()->getTimestamp(),
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
        $deleteResult = $this->collection->deleteOne(['state' => 'ny']);

        return $deleteResult->getDeletedCount() > 0;
    }

    public function countAll(array $scope = []): int
    {
        return $this->collection->countDocuments($scope);
    }

    public function findAll(array $scope = [], array $orderBy = []): iterable
    {
        $cursor = $this->collection->find($scope, [
            'sort' => $orderBy,
        ]);

        foreach ($cursor as $document) {
            yield $this->mapDocumentInfoEvent($document);
        }
    }

    public function findByPK(mixed $id): ?Event
    {
        return $this->findOne(['_id' => $id]);
    }

    public function findOne(array $scope = []): ?Event
    {
        $document = $this->collection->findOne($scope);

        if ($document === null) {
            return null;
        }

        return $this->mapDocumentInfoEvent($document);
    }

    public function mapDocumentInfoEvent(array $document): Event
    {
        return new Event(
            uuid: Uuid::fromString($document['_id']),
            type: $document['type'],
            payload: new Json($document['payload']),
            date: Carbon::createFromTimestamp($document['created_at'])->toDateTimeImmutable(),
            projectId: $document['project_id'],
        );
    }
}
