<?php

declare(strict_types=1);

namespace Modules\Projects\Interfaces\Commands;

use App\Application\Commands\CreateProject;
use Modules\Projects\Domain\Project;
use Modules\Projects\Domain\ProjectFactoryInterface;
use Modules\Projects\Domain\ProjectRepositoryInterface;
use Modules\Projects\Domain\ValueObject\Key;
use Modules\Projects\Exception\UniqueKeyException;
use Spiral\Cqrs\Attribute\CommandHandler;

final readonly class CreateProjectHandler
{
    public function __construct(
        private ProjectFactoryInterface $factory,
        private ProjectRepositoryInterface $projects,
    ) {}

    #[CommandHandler]
    public function __invoke(CreateProject $command): Project
    {
        if ($this->projects->findByPK($command->key) !== null) {
            throw new UniqueKeyException("Project with key {$command->key} already exists");
        }

        $project = $this->factory->create(
            key: Key::create($command->key),
            name: $command->name,
        );

        $this->projects->store($project);

        return $project;
    }
}
