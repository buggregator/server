<?php

declare(strict_types=1);

namespace Modules\Sentry\Application;

use Modules\Sentry\EventHandler;
use Psr\Container\ContainerInterface;
use Spiral\Boot\Bootloader\Bootloader;

final class SentryBootloader extends Bootloader
{
    protected const SINGLETONS = [
        EventHandlerInterface::class => [self::class, 'eventHandler'],
    ];

    public function eventHandler(ContainerInterface $container): EventHandlerInterface
    {
        return new EventHandler($container, []);
    }
}
