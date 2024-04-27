<?php

declare(strict_types=1);

namespace App\Integration\RoadRunner\Persistence;

use Doctrine\Common\Collections\ArrayCollection;
use Doctrine\Common\Collections\Criteria;
use Modules\Projects\Domain\Project;
use Modules\Projects\Domain\ProjectRepositoryInterface;
use Modules\Projects\Domain\ValueObject\Key;
use Modules\Projects\Exception\UniqueKeyException;
use Psr\SimpleCache\CacheInterface;

final readonly class CacheProjectRepository implements ProjectRepositoryInterface
{
    private const PROJECTS_KEY = 'projects';

    public function __construct(
        private CacheInterface $cache,
    ) {
    }

    public function store(Project $project): bool
    {
        $key = $project->getKey();

        $projects = $this->getProjects();
        $exists = $this->findByPK($key);
        if ($exists) {
            throw new UniqueKeyException(\sprintf('Project with key %s already exists', $key));
        }

        $projects[] = [
            'key' => (string)$project->getKey(),
            'name' => $project->getName(),
        ];

        $this->cache->set(self::PROJECTS_KEY, $projects);

        return true;
    }

    public function deleteByPK(string $key): bool
    {
        $projects = $this->getProjects();
        foreach ($projects as $key => $project) {
            if ($project['key'] === $key) {
                unset($projects[$key]);
                $this->cache->set(self::PROJECTS_KEY, $projects);

                return true;
            }
        }

        return false;
    }

    public function findByPK(mixed $key): ?object
    {
        $projects = $this->getProjects();

        foreach ($projects as $document) {
            if ($document['key'] === $key) {
                return $this->mapDocumentIntoProject($document);
            }
        }

        return null;
    }

    public function findOne(array $scope = []): ?object
    {
        $projects = $this->getProjects($scope);

        foreach ($projects as $document) {
            return $this->mapDocumentIntoProject($document);
        }

        return null;
    }

    public function findAll(array $scope = [], array $orderBy = [], int $limit = 30, int $offset = 0): iterable
    {
        $projects = $this->getProjects($scope, $orderBy);
        $projects = \array_slice($projects, $offset, $limit);

        foreach ($projects as $document) {
            yield $this->mapDocumentIntoProject($document);
        }
    }

    private function getProjects(array $scope = [], array $orderBy = []): array
    {
        $projects = $this->cache->get(self::PROJECTS_KEY, []);
        $criteria = (new Criteria())->orderBy($orderBy);
        foreach ($scope as $key => $value) {
            match (true) {
                \is_array($value) => $criteria->orWhere(Criteria::expr()->in($key, $value)),
                null === $value => $criteria->orWhere(Criteria::expr()->isNull($key)),
                default => $criteria->orWhere(Criteria::expr()->eq($key, $value)),
            };
        }

        return \array_keys((new ArrayCollection($projects))->matching($criteria)->toArray());
    }

    private function mapDocumentIntoProject(array $document): Project
    {
        return new Project(
            key: new Key($document['key']),
            name: $document['name'],
        );
    }
}
