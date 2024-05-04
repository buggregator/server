<?php

declare(strict_types=1);

namespace App\Integration\MongoDb\Persistence;

use App\Integration\MongoDb\Mappers\EventMapper;
use Modules\Events\Domain\Event;
use Modules\Events\Domain\EventRepositoryInterface;
use MongoDB\Collection;
use MongoDB\Model\BSONDocument;

final readonly class MongoDBEventRepository implements EventRepositoryInterface
{
    use HelpersTrait;

    public function __construct(
        private Collection $collection,
        private EventMapper $mapper,
    ) {}

    public function store(Event $event): bool
    {
        if ($this->findByPK($event->getUuid()) !== null) {
            $this->collection->replaceOne(
                ['_id' => (string) $event->getUuid()],
                $this->mapper->toDocument($event),
            );

            return true;
        }

        $result = $this->collection->insertOne($this->mapper->toDocument($event));

        return $result->getInsertedCount() > 0;
    }

    public function deleteAll(array $scope = []): void
    {
        $this->collection->deleteMany($this->buildScope($scope));
    }

    public function deleteByPK(string $uuid): bool
    {
        $deleteResult = $this->collection->deleteOne(['_id' => $uuid]);

        return $deleteResult->getDeletedCount() > 0;
    }

    public function countAll(array $scope = []): int
    {
        return $this->collection->countDocuments($this->buildScope($scope));
    }

    public function findAll(array $scope = [], array $orderBy = [], int $limit = 30, int $offset = 0): iterable
    {
        $cursor = $this->collection->find(
            $this->buildScope($scope),
            $this->makeOptions(
                orderBy: $orderBy,
                limit: $limit,
                offset: $offset,
            ),
        );

        foreach ($cursor as $document) {
            yield $this->mapper->toEvent($document);
        }
    }

    /**
     * @psalm-suppress ParamNameMismatch
     */
    public function findByPK(mixed $uuid): ?Event
    {
        return $this->findOne(['_id' => (string) $uuid]);
    }

    public function findOne(array $scope = []): ?Event
    {
        /** @var BSONDocument|null $document */
        $document = $this->collection->findOne($this->buildScope($scope));

        if ($document === null) {
            return null;
        }

        return $this->mapper->toEvent($document);
    }

    private function buildScope(array $scope): array
    {
        $newScope = [];

        foreach ($scope as $key => $value) {
            if ($key === 'uuid') {
                $key = '_id';
            }
            $newScope[$key] = \is_array($value) ? ['$in' => $value] : $value;
        }

        return $newScope;
    }
}
