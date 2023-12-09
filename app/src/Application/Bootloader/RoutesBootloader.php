<?php

declare(strict_types=1);

namespace App\Application\Bootloader;

use App\Application\HTTP\Middleware\JsonPayloadMiddleware;
use App\Interfaces\Http\EventHandlerAction;
use Spiral\Bootloader\Http\RoutesBootloader as BaseRoutesBootloader;
use Spiral\Filter\ValidationHandlerMiddleware;
use Spiral\Http\Middleware\ErrorHandlerMiddleware;
use Spiral\Router\Bootloader\AnnotatedRoutesBootloader;
use Spiral\Router\GroupRegistry;
use Spiral\Router\Loader\Configurator\RoutingConfigurator;
use Spiral\OpenApi\Controller\DocumentationController;

final class RoutesBootloader extends BaseRoutesBootloader
{
    protected const DEPENDENCIES = [
        AnnotatedRoutesBootloader::class,
    ];

    protected function globalMiddleware(): array
    {
        return [
            JsonPayloadMiddleware::class,
            ErrorHandlerMiddleware::class,
            ValidationHandlerMiddleware::class,
        ];
    }

    protected function middlewareGroups(): array
    {
        return [
            'web' => [],
            'api' => [],
            'docs' => [],
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
        $routes
            ->add('swagger-api-html', '/api/docs')
            ->group('docs')
            ->action(DocumentationController::class, 'html');

        $routes
            ->add('swagger-api-json', '/api/docs.json')
            ->group('docs')
            ->action(DocumentationController::class, 'json');

        $routes
            ->add('swagger-api-yaml', '/api/docs.yaml')
            ->group('docs')
            ->action(DocumentationController::class, 'yaml');

        $routes->default('/<path:.*>')
            ->group('web')
            ->action(EventHandlerAction::class, 'handle');
    }
}
