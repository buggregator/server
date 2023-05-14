<?php

declare(strict_types=1);

namespace Modules\HttpDumps\Application;

use Modules\HttpDumps\EventHandler;
use Psr\Container\ContainerInterface;
use Spiral\Boot\Bootloader\Bootloader;

final class HttpDumpsBootloader extends Bootloader
{
    protected const SINGLETONS = [
        EventHandlerInterface::class => [self::class, 'eventHandler'],
    ];

    public function eventHandler(ContainerInterface $container): EventHandlerInterface
    {
        return new EventHandler($container, []);
    }
}
