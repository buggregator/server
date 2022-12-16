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
use Spiral\Core\FactoryInterface;

final class PersistenceBootloader extends Bootloader
{
    protected const SINGLETONS = [
        EventRepositoryInterface::class => [self::class, 'createRepository'],
        CycleOrmEventRepository::class => [self::class, 'createCycleOrmEventRepository'],
        MongoDBEventRepository::class => [self::class, 'createMongoDBEventRepository'],
        CacheEventRepository::class => [self::class, 'createCacheEventRepository'],
    ];

    private function createCacheEventRepository(
        CacheStorageProviderInterface $provider,
    ): EventRepositoryInterface {
        return new CacheEventRepository($provider);
    }

    private function createRepository(
        FactoryInterface $factory,
        EnvironmentInterface $env
    ): EventRepositoryInterface {
        return match ($env->get('PERSISTENCE_DRIVER', 'cache')) {
            'cycle' => $factory->make(CycleOrmEventRepository::class),
            'mongodb' => $factory->make(MongoDBEventRepository::class),
            'cache' => $factory->make(CacheEventRepository::class),
            default => throw new \InvalidArgumentException('Unknown persistence driver'),
        };
    }

    private function createCycleOrmEventRepository(
        ORMInterface $orm,
        EntityManagerInterface $manager
    ): CycleOrmEventRepository {
        return new CycleOrmEventRepository($manager, new Select($orm, Event::class));
    }

    private function createMongoDBEventRepository(Database $database): MongoDBEventRepository
    {
        return new MongoDBEventRepository(
            $database->selectCollection('events')
        );
    }
}
