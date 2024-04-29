<?php

declare(strict_types=1);

namespace Modules\Projects\Domain;

use Cycle\ORM\RepositoryInterface;

/**
 * @extends RepositoryInterface<Project>
 */
interface ProjectRepositoryInterface extends RepositoryInterface
{
    public function store(Project $project): bool;

    public function deleteByPK(string $key): bool;
}
