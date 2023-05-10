<?php

declare(strict_types=1);

namespace Modules\Ray\Application;

use Modules\Ray\Application\Handlers\MergeEventsHandler;
use Modules\Ray\EventHandler;
use Psr\Container\ContainerInterface;
use Spiral\Boot\Bootloader\Bootloader;

final class RayBootloader extends Bootloader
{
    protected const SINGLETONS = [
        EventHandlerInterface::class => [self::class, 'eventHandler'],
    ];

    public function eventHandler(ContainerInterface $container): EventHandlerInterface
    {
        return new EventHandler($container, [
            MergeEventsHandler::class,
        ]);
    }
}
