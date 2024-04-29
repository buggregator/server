<?php

declare(strict_types=1);

namespace Modules\Projects\Interfaces\Console\Command;

use App\Application\Commands\CreateProject;
use Modules\Projects\Domain\ProjectLocatorInterface;
use Spiral\Console\Attribute\AsCommand;
use Spiral\Console\Command;
use Spiral\Cqrs\CommandBusInterface;

#[AsCommand(
    name: 'projects:register',
    description: 'Find and register all projects in the system'
)]
final class RegisterCommand extends Command
{
    public function __invoke(
        CommandBusInterface $bus,
        ProjectLocatorInterface $locator,
    ): int {
        foreach ($locator->findAll() as $project) {
            try {
                $this->writeln("Registering project: {$project->getName()}");
                $bus->dispatch(
                    new CreateProject(
                        key: (string)$project->getKey(),
                        name: $project->getName(),
                    ),
                );
            } catch (\Throwable $e) {
                $this->error($e->getMessage());
            }
        }

        return self::SUCCESS;
    }
}
