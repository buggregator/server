<?php

declare(strict_types=1);

namespace App\Application\Bootloader;

use Spiral\Core\Core;
use Psr\Container\ContainerInterface;
use Psr\EventDispatcher\EventDispatcherInterface;
use Spiral\Core\InterceptableCore;
use App\Application\AppVersion;
use App\Application\Client\Settings;
use App\Application\HTTP\Interceptor\JsonResourceInterceptor;
use App\Application\HTTP\Interceptor\StringToIntParametersInterceptor;
use App\Application\HTTP\Interceptor\UuidParametersConverterInterceptor;
use App\Application\Ide\UrlTemplate;
use App\Interfaces\Console\RegisterModulesCommand;
use Spiral\Boot\EnvironmentInterface;
use Spiral\Bootloader\DomainBootloader;
use Spiral\Console\Bootloader\ConsoleBootloader;
use Spiral\Core\CoreInterface;

final class AppBootloader extends DomainBootloader
{
    public function defineSingletons(): array
    {
        return [
            CoreInterface::class => fn(
                Core $core,
                ContainerInterface $container,
                ?EventDispatcherInterface $dispatcher = null,
            ): InterceptableCore => self::domainCore($core, $container, $dispatcher),

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

            Settings::class => fn(
                EnvironmentInterface $env,
            ): AppVersion => new AppVersion(
                version: $env->get('APP_VERSION', 'dev'),
            ),

            ClientSettings::class => fn(
                EnvironmentInterface $env,
            ): ClientSettings => new ClientSettings(
                version: $env->get('CLIENT_SUPPORTED_EVENTS', 'http-dump,inspector,monolog,profiler,ray,sentry,smtp,var-dump'),
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

    public function init(ConsoleBootloader $console): void
    {
        $console->addSequence(
            name: RegisterModulesCommand::SEQUENCE,
            sequence: 'database:check-connection',
            header: 'Check database connection',
        );
    }
}
