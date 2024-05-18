<?php

declare(strict_types=1);

namespace Modules\Projects\Application;

use App\Application\Persistence\DriverEnum;
use App\Interfaces\Console\RegisterModulesCommand;
use Cycle\ORM\EntityManagerInterface;
use Cycle\ORM\ORMInterface;
use Cycle\ORM\Select;
use Modules\Projects\Domain\Project;
use Modules\Projects\Domain\ProjectFactoryInterface;
use Modules\Projects\Domain\ProjectLocatorInterface;
use Modules\Projects\Domain\ProjectRepositoryInterface;
use Modules\Projects\Integration\CycleOrm\ProjectRepository;
use Spiral\Boot\Bootloader\Bootloader;
use Spiral\Boot\DirectoriesInterface;
use Spiral\Console\Bootloader\ConsoleBootloader;
use Spiral\Core\FactoryInterface;

final class ProjectBootloader extends Bootloader
{
    public function defineSingletons(): array
    {
        return [
            ProjectFactoryInterface::class => ProjectFactory::class,

            YamlFileProjectLocator::class => static fn(
                FactoryInterface $factory,
                DirectoriesInterface $dirs,
            ): YamlFileProjectLocator => $factory->make(
                YamlFileProjectLocator::class,
                [
                    'directory' => $dirs->get('runtime') . '/configs',
                ],
            ),

            ProjectLocatorInterface::class => static function (
                YamlFileProjectLocator $locator,
            ): ProjectLocatorInterface {
                return new CompositeProjectLocator([
                    $locator,
                ]);
            },

            ProjectRepositoryInterface::class => static fn(
                FactoryInterface $factory,
                DriverEnum $driver,
            ): ProjectRepositoryInterface => match ($driver) {
                DriverEnum::Database => $factory->make(ProjectRepository::class),
                default => throw new \Exception('Unsupported database driver'),
            },
            ProjectRepository::class => static fn(
                ORMInterface $orm,
                EntityManagerInterface $manager,
            ): ProjectRepositoryInterface => new ProjectRepository(
                $manager,
                new Select($orm, Project::class),
            ),
        ];
    }

    public function init(ConsoleBootloader $console): void
    {
        $console->addSequence(
            name: RegisterModulesCommand::SEQUENCE,
            sequence: 'projects:register',
            header: 'Register all projects in the system',
        );
    }
}
