<?php

declare(strict_types=1);

namespace App\Application\Bootloader;

use App\Application\Persistence\DriverEnum;
use App\Integration\CycleOrm\Persistence\CycleOrmEventRepository;
use App\Integration\CycleOrm\Persistence\CycleOrmProjectRepository;
use App\Integration\MongoDb\Persistence\MongoDBEventRepository;
use App\Integration\MongoDb\Persistence\MongoDBProjectRepository;
use App\Integration\RoadRunner\Persistence\CacheEventRepository;
use App\Integration\RoadRunner\Persistence\CacheProjectRepository;
use Cycle\ORM\EntityManagerInterface;
use Cycle\ORM\ORMInterface;
use Cycle\ORM\Select;
use Modules\Events\Domain\Event;
use Modules\Events\Domain\EventRepositoryInterface;
use Modules\Projects\Domain\Project;
use Modules\Projects\Domain\ProjectRepositoryInterface;
use MongoDB\Database;
use Spiral\Boot\Bootloader\Bootloader;
use Spiral\Boot\EnvironmentInterface;
use Spiral\Cache\CacheStorageProviderInterface;
use Spiral\Console\Bootloader\ConsoleBootloader;
use Spiral\Core\FactoryInterface;
use Spiral\Cycle\Bootloader as CycleBridge;

final class PersistenceBootloader extends Bootloader
{
    public function defineDependencies(): array
    {
        return [
            // Databases
            CycleBridge\DatabaseBootloader::class,
            CycleBridge\MigrationsBootloader::class,

            // ORM
            CycleBridge\SchemaBootloader::class,
            CycleBridge\CycleOrmBootloader::class,
            CycleBridge\AnnotatedBootloader::class,
        ];
    }

    public function defineSingletons(): array
    {
        return [
            // Events
            EventRepositoryInterface::class => static fn(
                FactoryInterface $factory,
                DriverEnum $driver,
            ): EventRepositoryInterface => match ($driver) {
                DriverEnum::Database => $factory->make(CycleOrmEventRepository::class),
                DriverEnum::MongoDb => $factory->make(MongoDBEventRepository::class),
                DriverEnum::InMemory => $factory->make(CacheEventRepository::class),
            },
            CycleOrmEventRepository::class => static fn(
                ORMInterface $orm,
                EntityManagerInterface $manager,
            ): CycleOrmEventRepository => new CycleOrmEventRepository($manager, new Select($orm, Event::class)),
            MongoDBEventRepository::class => static fn(
                Database $database,
            ): MongoDBEventRepository => new MongoDBEventRepository(
                $database->selectCollection('events'),
            ),
            CacheEventRepository::class => static fn(
                CacheStorageProviderInterface $provider,
                EnvironmentInterface $env,
            ): EventRepositoryInterface => new CacheEventRepository(
                cache: $provider->storage('events'),
                ttl: (int)$env->get('EVENTS_CACHE_TTL', 60 * 60 * 2),
            ),

            // Projects
            ProjectRepositoryInterface::class => static fn(
                FactoryInterface $factory,
                DriverEnum $driver,
            ): ProjectRepositoryInterface => match ($driver) {
                DriverEnum::Database => $factory->make(CycleOrmProjectRepository::class),
                DriverEnum::MongoDb => $factory->make(MongoDBProjectRepository::class),
                DriverEnum::InMemory => $factory->make(CacheProjectRepository::class),
            },
            CycleOrmProjectRepository::class => static fn(
                ORMInterface $orm,
                EntityManagerInterface $manager,
            ): ProjectRepositoryInterface => new CycleOrmProjectRepository($manager, new Select($orm, Project::class)),
            MongoDBProjectRepository::class => static fn(
                Database $database,
            ): ProjectRepositoryInterface => new MongoDBProjectRepository(
                $database->selectCollection('projects'),
            ),
            CacheProjectRepository::class => static fn(
                CacheStorageProviderInterface $provider,
            ): ProjectRepositoryInterface => new CacheProjectRepository(
                cache: $provider->storage('projects'),
            ),
        ];
    }

    public function init(ConsoleBootloader $console, DriverEnum $driver): void
    {
        if ($driver === DriverEnum::Database) {
            $console->addConfigureSequence(
                sequence: 'migrate',
                header: 'Migration',
            );
        }
    }
}
