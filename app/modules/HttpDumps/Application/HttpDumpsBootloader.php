<?php

declare(strict_types=1);

namespace Modules\HttpDumps\Application;

use Modules\HttpDumps\EventHandler;
use Psr\Container\ContainerInterface;
use Spiral\Boot\Bootloader\Bootloader;

final class HttpDumpsBootloader extends Bootloader
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
