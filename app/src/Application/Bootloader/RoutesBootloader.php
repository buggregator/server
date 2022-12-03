<?php

declare(strict_types=1);

namespace App\Application\Bootloader;

use Spiral\Auth\Middleware\AuthTransportMiddleware;
use Spiral\Bootloader\Http\RoutesBootloader as BaseRoutesBootloader;
use Spiral\Core\Container\Autowire;
use Spiral\Http\Middleware\ErrorHandlerMiddleware;
use Spiral\Http\Middleware\JsonPayloadMiddleware;
use Spiral\Router\Loader\Configurator\RoutingConfigurator;

final class RoutesBootloader extends BaseRoutesBootloader
{
    protected function globalMiddleware(): array
    {
        return [
            ErrorHandlerMiddleware::class,
            JsonPayloadMiddleware::class,
        ];
    }

    protected function middlewareGroups(): array
    {
        return [
            'api' => [
                new Autowire(AuthTransportMiddleware::class, ['transportName' => 'header']),
            ],
        ];
    }

    protected function defineRoutes(RoutingConfigurator $routes): void
    {
    }
}
