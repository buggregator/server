<?php

declare(strict_types=1);

namespace Modules\Sentry\Application;

use Modules\Sentry\EventHandler;
use Psr\Container\ContainerInterface;
use Spiral\Boot\Bootloader\Bootloader;
use Spiral\Boot\EnvironmentInterface;

final class SentryBootloader extends Bootloader
{
    public function defineSingletons(): array
    {
        return [
            SecretKeyValidator::class => static fn(
                EnvironmentInterface $env,
            ): SecretKeyValidator => new SecretKeyValidator(
                secret: $env->get('SENTRY_SECRET_KEY'),
            ),

            EventHandlerInterface::class => static fn(
                ContainerInterface $container,
            ): EventHandlerInterface => new EventHandler($container, []),
        ];
    }
}
