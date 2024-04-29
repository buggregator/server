<?php

declare(strict_types=1);

namespace App\Application\Bootloader;

use App\Application\Broadcasting\EventMapper;
use App\Application\Broadcasting\EventMapperInterface;
use App\Application\Broadcasting\EventMapperRegistryInterface;
use App\Application\Event\EventTypeMapper;
use App\Application\Event\EventTypeMapperInterface;
use App\Application\Event\EventTypeRegistryInterface;
use Spiral\Boot\Bootloader\Bootloader;
use Spiral\Cqrs\Bootloader\CqrsBootloader;
use Spiral\Events\Bootloader\EventsBootloader;
use Spiral\League\Event\Bootloader\EventBootloader;
use Spiral\RoadRunnerBridge\Bootloader\CentrifugoBootloader;

final class BroadcastingBootloader extends Bootloader
{
    public function defineDependencies(): array
    {
        return [
            CentrifugoBootloader::class,
            EventsBootloader::class,
            EventBootloader::class,
            CqrsBootloader::class,
        ];
    }

    public function defineSingletons(): array
    {
        return [
            EventMapper::class => EventMapper::class,
            EventMapperInterface::class => EventMapper::class,
            EventMapperRegistryInterface::class => EventMapper::class,

            EventTypeMapper::class => EventTypeMapper::class,
            EventTypeMapperInterface::class => EventTypeMapper::class,
            EventTypeRegistryInterface::class => EventTypeMapper::class,
        ];
    }
}
