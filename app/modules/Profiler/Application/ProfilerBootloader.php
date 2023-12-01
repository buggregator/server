<?php

declare(strict_types=1);

namespace Modules\Profiler\Application;

use Modules\Profiler\Application\Handlers\CalculateDiffsBetweenEdges;
use Modules\Profiler\Application\Handlers\CleanupEvent;
use Modules\Profiler\Application\Handlers\PrepareEdges;
use Modules\Profiler\Application\Handlers\PreparePeaks;
use Psr\Container\ContainerInterface;
use Spiral\Boot\Bootloader\Bootloader;

final class ProfilerBootloader extends Bootloader
{
    public function defineSingletons(): array
    {
        return [
            EventHandlerInterface::class => static function (ContainerInterface $container): EventHandlerInterface {
                return new EventHandler($container, [
                    PreparePeaks::class,
                    CalculateDiffsBetweenEdges::class,
                    PrepareEdges::class,
                    CleanupEvent::class,
                ]);
            },
        ];
    }
}
