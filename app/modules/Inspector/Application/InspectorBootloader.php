<?php

declare(strict_types=1);

namespace Modules\Inspector\Application;

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
}
