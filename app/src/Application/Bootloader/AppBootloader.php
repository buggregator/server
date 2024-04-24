<?php

declare(strict_types=1);

namespace App\Application\Bootloader;

use App\Application\AppVersion;
use App\Application\HTTP\Interceptor\JsonResourceInterceptor;
use App\Application\HTTP\Interceptor\StringToIntParametersInterceptor;
use App\Application\HTTP\Interceptor\UuidParametersConverterInterceptor;
use App\Application\Ide\UrlTemplate;
use Spiral\Boot\EnvironmentInterface;
use Spiral\Bootloader\DomainBootloader;
use Spiral\Core\CoreInterface;

final class AppBootloader extends DomainBootloader
{
    public function defineSingletons(): array
    {
        return [
            CoreInterface::class => fn(
                \Spiral\Core\Core $core,
                \Psr\Container\ContainerInterface $container,
                ?\Psr\EventDispatcher\EventDispatcherInterface $dispatcher = null,
            ): \Spiral\Core\InterceptableCore => self::domainCore($core, $container, $dispatcher),

            UrlTemplate::class => fn(
                EnvironmentInterface $env,
            ): UrlTemplate => new UrlTemplate(
                template: $env->get('IDE_URL_TEMPLATE', 'phpstorm://open?url=file://%s&line=%d'),
            ),

            AppVersion::class => fn(
                EnvironmentInterface $env,
            ): AppVersion => new AppVersion(
                version: $env->get('APP_VERSION', 'dev'),
            ),
        ];
    }

    protected static function defineInterceptors(): array
    {
        return [
            StringToIntParametersInterceptor::class,
            UuidParametersConverterInterceptor::class,
            JsonResourceInterceptor::class,
        ];
    }
}
