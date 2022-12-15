<?php

declare(strict_types=1);

namespace App\Application\Bootloader;

use Spiral\Auth\Middleware\AuthTransportMiddleware;
use Spiral\Bootloader\Http\RoutesBootloader as BaseRoutesBootloader;
use Spiral\Core\Container\Autowire;
use Spiral\Filter\ValidationHandlerMiddleware;
use Spiral\Http\Middleware\ErrorHandlerMiddleware;
use Spiral\Http\Middleware\JsonPayloadMiddleware;
use Spiral\Router\Bootloader\AnnotatedRoutesBootloader;
use Spiral\Router\GroupRegistry;
use Spiral\Router\Loader\Configurator\RoutingConfigurator;

final class RoutesBootloader extends BaseRoutesBootloader
{
    protected const DEPENDENCIES = [
        AnnotatedRoutesBootloader::class,
    ];

    protected function globalMiddleware(): array
    {
        return [
            ErrorHandlerMiddleware::class,
            ValidationHandlerMiddleware::class,
            //JsonPayloadMiddleware::class,
        ];
    }

    protected function middlewareGroups(): array
    {
        return [
            'api' => [
                //new Autowire(AuthTransportMiddleware::class, ['transportName' => 'header']),
            ],
        ];
    }

    /**
     * Override this method to configure route groups
     */
    protected function configureRouteGroups(GroupRegistry $groups): void
    {
        $groups->getGroup('api')->setPrefix('api/');
    }
}
