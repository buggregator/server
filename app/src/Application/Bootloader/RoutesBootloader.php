<?php

declare(strict_types=1);

namespace App\Application\Bootloader;

use App\Application\Service\HttpHandler\CoreHandlerInterface;
use App\Interfaces\Http\EventHandlerAction;
use Psr\Container\ContainerInterface;
use Psr\Http\Message\ResponseInterface;
use Psr\Http\Message\ServerRequestInterface;
use Spiral\Bootloader\Http\RoutesBootloader as BaseRoutesBootloader;
use Spiral\Filter\ValidationHandlerMiddleware;
use Spiral\Http\Middleware\ErrorHandlerMiddleware;
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
        ];
    }

    protected function middlewareGroups(): array
    {
        return [
            'api' => [],
        ];
    }

    /**
     * Override this method to configure route groups
     */
    protected function configureRouteGroups(GroupRegistry $groups): void
    {
        $groups->getGroup('api')->setPrefix('api/');
    }

    protected function defineRoutes(RoutingConfigurator $routes): void
    {
        $routes->default('/<path:.*>')
            ->group('web')
            ->action(EventHandlerAction::class, 'handle');
    }
}
