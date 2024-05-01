<?php

declare(strict_types=1);

namespace App\Application\Bootloader;

use App\Application\Auth\AuthSettings;
use App\Application\HTTP\Middleware\ApiAuthMiddleware;
use App\Application\HTTP\Middleware\DetectEventTypeMiddleware;
use App\Application\HTTP\Middleware\JsonPayloadMiddleware;
use App\Interfaces\Http\EventHandlerAction;
use Spiral\Auth\Middleware\Firewall\ExceptionFirewall;
use Spiral\Bootloader\Http\RoutesBootloader as BaseRoutesBootloader;
use Spiral\Core\Container;
use Spiral\Filter\ValidationHandlerMiddleware;
use Spiral\Http\Exception\ClientException\ForbiddenException;
use Spiral\Http\Middleware\ErrorHandlerMiddleware;
use Spiral\Nyholm\Bootloader\NyholmBootloader;
use Spiral\OpenApi\Bootloader\SwaggerBootloader;
use Spiral\Router\Bootloader\AnnotatedRoutesBootloader;
use Spiral\Router\GroupRegistry;
use Spiral\Router\Loader\Configurator\RoutingConfigurator;
use Spiral\OpenApi\Controller\DocumentationController;
use Spiral\Session\Middleware\SessionMiddleware;
use Spiral\Bootloader as Framework;

final class RoutesBootloader extends BaseRoutesBootloader
{
    public function __construct(
        private readonly Container $container,
    ) {}

    public function defineDependencies(): array
    {
        return [
            NyholmBootloader::class,
            Framework\Http\RouterBootloader::class,
            Framework\Http\JsonPayloadsBootloader::class,
            Framework\Http\SessionBootloader::class,
            SwaggerBootloader::class,
            AnnotatedRoutesBootloader::class,
        ];
    }

    protected function globalMiddleware(): array
    {
        return [
            SessionMiddleware::class,
            JsonPayloadMiddleware::class,
            ErrorHandlerMiddleware::class,
            ValidationHandlerMiddleware::class,
        ];
    }

    protected function middlewareGroups(): array
    {
        return [
            'auth' => [
                ApiAuthMiddleware::class,
            ],
            'guest' => [
                'middleware:auth',
            ],
            'api_guest' => [
                'middleware:auth',
            ],
            'web' => [
                'middleware:auth',
            ],
            'web_guest' => [
                'middleware:auth',
            ],
            'api' => [
                'middleware:auth',
            ],
            'docs' => [],
        ];
    }

    /**
     * Override this method to configure route groups
     */
    protected function configureRouteGroups(GroupRegistry $groups): void
    {
        $groups->getGroup('api')->setPrefix('api/');
        $groups->getGroup('api_guest')->setPrefix('api/');

        $settings = $this->container->get(AuthSettings::class);

        if ($settings->enabled) {
            $groups->getGroup('api')
                ->addMiddleware(new ExceptionFirewall(new ForbiddenException()));
            $groups->getGroup('web')
                ->addMiddleware(new ExceptionFirewall(new ForbiddenException()));
        }
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
            ->group('web_guest')
            ->middleware(DetectEventTypeMiddleware::class)
            ->action(EventHandlerAction::class, 'handle');
    }
}
