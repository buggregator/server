<?php

declare(strict_types=1);

namespace Modules\Projects\Interfaces\Queries;

use App\Application\Commands\FindProjectByKey;
use Modules\Projects\Domain\Project;
use Modules\Projects\Domain\ProjectRepositoryInterface;
use Spiral\Cqrs\Attribute\QueryHandler;

final readonly class FindProjectByKeyHandler
{
    public function __construct(
        private ProjectRepositoryInterface $projects,
    ) {}

    #[QueryHandler]
    public function __invoke(FindProjectByKey $query): ?Project
    {
        return $this->projects->findByPK($query->key);
    }
}
