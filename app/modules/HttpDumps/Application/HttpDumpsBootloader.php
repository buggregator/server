<?php

declare(strict_types=1);

namespace Modules\HttpDumps\Application;

use App\Application\Service\HttpHandler\HandlerRegistryInterface;
use Modules\HttpDumps\EventHandler;
use Modules\HttpDumps\Interfaces\Http\Handler\AnyHttpRequestDump;
use Psr\Container\ContainerInterface;
use Spiral\Boot\Bootloader\Bootloader;

final class HttpDumpsBootloader extends Bootloader
{
    protected const SINGLETONS = [
        EventHandlerInterface::class => [self::class, 'eventHandler'],
    ];

    public function boot(
        HandlerRegistryInterface $registry,
        AnyHttpRequestDump $handler
    ): void {
        $registry->register($handler);
    }

    public function eventHandler(ContainerInterface $container): EventHandlerInterface
    {
        return new EventHandler($container, []);
    }
}
