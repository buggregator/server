<?php

declare(strict_types=1);

namespace Modules\Projects\Interfaces\Console\Command;

use Modules\Projects\Domain\ProjectLocatorInterface;
use Modules\Projects\Domain\ProjectRepositoryInterface;
use Spiral\Console\Attribute\AsCommand;
use Spiral\Console\Command;

#[AsCommand(
    name: 'projects:register',
    description: 'Find and register all projects in the system'
)]
final class RegisterCommand extends Command
{
    public function __invoke(
        ProjectRepositoryInterface $projects,
        ProjectLocatorInterface $locator,
    ): int {
        foreach ($locator->findAll() as $project) {
            if ($projects->findByPK($project->getKey()) !== null) {
                $this->writeln("Project already registered: {$project->getName()}");
                continue;
            }

            $this->writeln("Registering project: {$project->getName()}");
            $projects->store($project);
        }

        return self::SUCCESS;
    }
}
