<?php

declare(strict_types=1);

namespace Modules\Profiler\Application;

use App\Application\Event\EventTypeRegistryInterface;
use Modules\Profiler\Application\Handlers\CalculateDiffsBetweenEdges;
use Modules\Profiler\Application\Handlers\CleanupEvent;
use Modules\Profiler\Application\Handlers\PrepareEdges;
use Modules\Profiler\Application\Handlers\PreparePeaks;
use Modules\Profiler\Application\Handlers\StoreProfile;
use Modules\Profiler\Domain\EdgeFactoryInterface;
use Modules\Profiler\Domain\ProfileFactoryInterface;
use Modules\Profiler\Integration\CycleOrm\EdgeFactory;
use Modules\Profiler\Integration\CycleOrm\ProfileFactory;
use Modules\Profiler\Interfaces\Queries\FindCallGraphByUuidHandler;
use Modules\Profiler\Interfaces\Queries\FindFlameChartByUuidHandler;
use Psr\Container\ContainerInterface;
use Spiral\Boot\Bootloader\Bootloader;
use Spiral\Core\FactoryInterface;
use Spiral\Queue\QueueConnectionProviderInterface;
use Spiral\Storage\StorageInterface;

final class ProfilerBootloader extends Bootloader
{
    public function defineSingletons(): array
    {
        return [
            ProfileFactoryInterface::class => ProfileFactory::class,
            EdgeFactoryInterface::class => EdgeFactory::class,
            EventHandlerInterface::class => static fn(
                ContainerInterface $container,
            ): EventHandlerInterface => new EventHandler($container, [
                PreparePeaks::class,
                CalculateDiffsBetweenEdges::class,
                PrepareEdges::class,
                CleanupEvent::class,
                StoreProfile::class,
            ]),


            StoreProfile::class => static fn(
                FactoryInterface $factory,
                QueueConnectionProviderInterface $provider,
            ): StoreProfile => $factory->make(StoreProfile::class, [
                'queue' => $provider->getConnection('profiler'),
            ]),

            FindCallGraphByUuidHandler::class => FindCallGraphByUuidHandler::class,

            FindFlameChartByUuidHandler::class => static fn(
                FactoryInterface $factory,
                StorageInterface $storage,
            ): FindFlameChartByUuidHandler => $factory->make(FindFlameChartByUuidHandler::class, [
                'bucket' => $storage->bucket('profiles'),
            ]),
        ];
    }

    public function boot(EventTypeRegistryInterface $registry): void
    {
        $registry->register('profiler', new Mapper\EventTypeMapper());
    }
}
