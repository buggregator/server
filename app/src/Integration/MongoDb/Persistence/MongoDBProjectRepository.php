<?php

declare(strict_types=1);

namespace App\Integration\MongoDb\Persistence;

use App\Integration\MongoDb\Mappers\ProjectMapper;
use Modules\Projects\Domain\Project;
use Modules\Projects\Domain\ProjectRepositoryInterface;
use MongoDB\Collection;
use MongoDB\Model\BSONDocument;

final readonly class MongoDBProjectRepository implements ProjectRepositoryInterface
{
    use HelpersTrait;

    public function __construct(
        private Collection $collection,
        private ProjectMapper $mapper,
    ) {}

    public function store(Project $project): bool
    {
        $result = $this->collection->insertOne(
            $this->mapper->toDocument($project),
        );

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
        /** @var BSONDocument|null $document */
        $document = $this->collection->findOne($scope);

        if ($document === null) {
            return null;
        }

        return $this->mapper->toProject($document);
    }

    public function findAll(array $scope = [], array $orderBy = [], int $limit = 30, int $offset = 0): iterable
    {
        $cursor = $this->collection->find(
            $scope,
            $this->makeOptions(
                orderBy: $orderBy,
                limit: $limit,
                offset: $offset,
            ),
        );

        foreach ($cursor as $document) {
            yield $this->mapper->toProject($document);
        }
    }
}
