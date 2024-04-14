<?php

declare(strict_types=1);

namespace App\Application\Bootloader;

use App\Application\Persistence\CacheEventRepository;
use App\Application\Persistence\CycleOrmEventRepository;
use App\Application\Persistence\MongoDBEventRepository;
use Cycle\ORM\EntityManagerInterface;
use Cycle\ORM\ORMInterface;
use Cycle\ORM\Select;
use Modules\Events\Domain\Event;
use Modules\Events\Domain\EventRepositoryInterface;
use MongoDB\Database;
use Spiral\Boot\Bootloader\Bootloader;
use Spiral\Boot\EnvironmentInterface;
use Spiral\Cache\CacheStorageProviderInterface;
use Spiral\Console\Bootloader\ConsoleBootloader;
use Spiral\Core\FactoryInterface;

final class PersistenceBootloader extends Bootloader
{
    public function defineSingletons(): array
    {
        return [
            EventRepositoryInterface::class => static fn(
                FactoryInterface $factory,
                EnvironmentInterface $env,
            ): EventRepositoryInterface => match ($env->get('PERSISTENCE_DRIVER', 'cache')) {
                'cycle', 'database', 'db' => $factory->make(CycleOrmEventRepository::class),
                'mongodb', 'mongo' => $factory->make(MongoDBEventRepository::class),
                'cache', 'memory' => $factory->make(CacheEventRepository::class),
                default => throw new \InvalidArgumentException('Unknown persistence driver'),
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
            ): EventRepositoryInterface => new CacheEventRepository($provider),
        ];
    }

    public function init(ConsoleBootloader $console, EventRepositoryInterface $repository): void
    {
        if ($repository instanceof CycleOrmEventRepository) {
            $console->addConfigureSequence(
                sequence: 'migrate',
                header: 'Migration',
            );
        }
    }
}
