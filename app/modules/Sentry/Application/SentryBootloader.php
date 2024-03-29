<?php

declare(strict_types=1);

namespace Modules\Sentry\Application;

use Modules\Sentry\EventHandler;
use Psr\Container\ContainerInterface;
use Spiral\Boot\Bootloader\Bootloader;

final class SentryBootloader extends Bootloader
{
    public function defineSingletons(): array
    {
        return [
            EventHandlerInterface::class => static function (ContainerInterface $container): EventHandlerInterface {
                return new EventHandler($container, []);
            },
        ];
    }
}
