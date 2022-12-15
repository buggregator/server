<?php

declare(strict_types=1);

namespace Modules\Profiler\Application;

use Modules\Profiler\Application\Handlers\CleanupEvent;
use Modules\Profiler\Application\Handlers\PrepareEdges;
use Modules\Profiler\Application\Handlers\PreparePeaks;
use Psr\Container\ContainerInterface;
use Spiral\Boot\Bootloader\Bootloader;

final class ProfilerBootloader extends Bootloader
{
    protected const SINGLETONS = [
        EventHandlerInterface::class => [self::class, 'eventHandler'],
    ];

    public function eventHandler(ContainerInterface $container): EventHandlerInterface
    {
        return new EventHandler($container, [
            PrepareEdges::class,
            PreparePeaks::class,
            CleanupEvent::class,
        ]);
    }
}
