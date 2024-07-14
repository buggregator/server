<?php

declare(strict_types=1);

namespace Modules\Inspector\Application;

use Modules\Inspector\Application\Mapper\EventTypeMapper;
use App\Application\Event\EventTypeRegistryInterface;
use Spiral\Boot\Bootloader\Bootloader;
use Spiral\Boot\EnvironmentInterface;

final class InspectorBootloader extends Bootloader
{
    public function defineSingletons(): array
    {
        return [
            SecretKeyValidator::class => static fn(
                EnvironmentInterface $env,
            ): SecretKeyValidator => new SecretKeyValidator(
                secret: $env->get('INSPECTOR_SECRET_KEY'),
            ),
        ];
    }

    public function boot(EventTypeRegistryInterface $registry): void
    {
        $registry->register('inspector', new EventTypeMapper());
    }
}
