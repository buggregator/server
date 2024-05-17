<?php

declare(strict_types=1);

namespace Modules\Projects\Integration\CycleOrm;

use Cycle\ORM\EntityManagerInterface;
use Cycle\ORM\Select;
use Cycle\ORM\Select\Repository;
use Modules\Projects\Domain\Project;
use Modules\Projects\Domain\ProjectRepositoryInterface;

/**
 * @template TEntity of Project
 * @extends Repository<Project>
 */
final class ProjectRepository extends Repository implements ProjectRepositoryInterface
{
    public function __construct(
        private readonly EntityManagerInterface $entityManager,
        Select $select,
    ) {
        parent::__construct($select);
    }

    public function store(Project $project): bool
    {
        $this->entityManager->persist($project);
        $this->entityManager->run();

        return true;
    }

    public function deleteByPK(string $key): bool
    {
        $project = $this->findByPK($key);

        if (!$project) {
            return false;
        }

        $this->entityManager->delete($project);
        $this->entityManager->run();

        return true;
    }
}
