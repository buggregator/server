<?php

declare(strict_types=1);

namespace Modules\Profiler\Application;

use App\Application\Service\HttpHandler\HandlerRegistryInterface;
use Modules\Profiler\Application\Handlers\CalculateDiffsBetweenEdges;
use Modules\Profiler\Interfaces\Http\Handler\EventHandler as HttpEventHandler;
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

    public function boot(
        HandlerRegistryInterface $registry,
        HttpEventHandler $handler
    ): void {
        $registry->register($handler);
    }

    public function eventHandler(ContainerInterface $container): EventHandlerInterface
    {
        return new EventHandler($container, [
            PreparePeaks::class,
            CalculateDiffsBetweenEdges::class,
            PrepareEdges::class,
            CleanupEvent::class,
        ]);
    }
}
