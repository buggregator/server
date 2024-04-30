<?php

declare(strict_types=1);

namespace Modules\Projects\Interfaces\Queries;

use App\Application\Commands\FindAllProjects;
use Modules\Projects\Domain\ProjectRepositoryInterface;
use Spiral\Cqrs\Attribute\QueryHandler;

final readonly class FindProjectsHandler
{
    public function __construct(
        private ProjectRepositoryInterface $projects,
    ) {}

    #[QueryHandler]
    public function __invoke(FindAllProjects $query): iterable
    {
        return $this->projects->findAll();
    }
}
