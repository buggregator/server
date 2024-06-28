<?php

declare(strict_types=1);

namespace App\Application\Bootloader;

use Spiral\Core\Core;
use Psr\Container\ContainerInterface;
use Psr\EventDispatcher\EventDispatcherInterface;
use Spiral\Core\InterceptableCore;
use App\Application\AppVersion;
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
            CoreInterface::class => static fn(Core $core, ContainerInterface $container, ?EventDispatcherInterface $dispatcher = null): InterceptableCore => self::domainCore($core, $container, $dispatcher),

            UrlTemplate::class => static fn(EnvironmentInterface $env): UrlTemplate => new UrlTemplate(
                template: $env->get('IDE_URL_TEMPLATE', 'phpstorm://open?url=file://%s&line=%d'),
            ),

            AppVersion::class => static fn(EnvironmentInterface $env): AppVersion => new AppVersion(
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

    public function init(ConsoleBootloader $console): void
    {
        $console->addSequence(
            name: RegisterModulesCommand::SEQUENCE,
            sequence: 'database:check-connection',
            header: 'Check database connection',
        );
    }
}
