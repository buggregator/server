<?php

declare(strict_types=1);

namespace Modules\Ray\Application;

use App\Application\Service\HttpHandler\HandlerRegistryInterface;
use Modules\Ray\Application\Handlers\MergeEventsHandler;
use Modules\Ray\EventHandler;
use Modules\Ray\Interfaces\Http\Handler\EventHandler as HttpEventHandler;
use Psr\Container\ContainerInterface;
use Spiral\Boot\Bootloader\Bootloader;

final class RayBootloader extends Bootloader
{
    public function defineSingletons(): array
    {
        return [
            EventHandlerInterface::class => static fn(
                ContainerInterface $container,
            ): EventHandlerInterface => new EventHandler($container, [
                MergeEventsHandler::class,
            ]),
        ];
    }

    public function boot(
        HandlerRegistryInterface $registry,
        HttpEventHandler $handler,
    ): void {
        $registry->register($handler);
    }
}
