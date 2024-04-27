<?php

declare(strict_types=1);

namespace App\Integration\MongoDb\Persistence;

use App\Application\Domain\ValueObjects\Uuid;
use Modules\Projects\Domain\Project;
use Modules\Projects\Domain\ProjectRepositoryInterface;
use Modules\Projects\Domain\ValueObject\Key;
use MongoDB\Collection;
use MongoDB\Model\BSONDocument;

final readonly class MongoDBProjectRepository implements ProjectRepositoryInterface
{
    public function __construct(
        private Collection $collection,
    ) {
    }

    public function store(Project $project): bool
    {
        $result = $this->collection->insertOne([
            '_id' => (string) $project->getKey(),
            'name' => $project->getName(),
        ]);

        return $result->getInsertedCount() > 0;
    }

    public function deleteByPK(string $key): bool
    {
        $deleteResult = $this->collection->deleteOne(['_id' => $key]);

        return $deleteResult->getDeletedCount() > 0;
    }

    public function findByPK(mixed $key): ?object
    {
        return $this->findOne(['_id' => $key]);
    }

    public function findOne(array $scope = []): ?object
    {
        $document = $this->collection->findOne($scope);

        if ($document === null) {
            return null;
        }

        return $this->mapDocumentIntoProject($document);
    }

    public function findAll(array $scope = [], array $orderBy = [], int $limit = 30, int $offset = 0): iterable
    {
        $cursor = $this->collection->find($scope, [
            'sort' => $this->mapOrderBy($orderBy),
            'limit' => $limit,
            'skip' => $offset,
        ]);

        foreach ($cursor as $document) {
            yield $this->mapDocumentIntoProject($document);
        }
    }

    public function mapDocumentIntoProject(BSONDocument $document): Project
    {
        return new Project(
            key: new Key($document['_id']),
            name: $document['name'],
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
