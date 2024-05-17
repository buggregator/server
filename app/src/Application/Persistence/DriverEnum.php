<?php

namespace App\Application\Persistence;

use Spiral\Boot\EnvironmentInterface;
use Spiral\Boot\Injector\InjectableEnumInterface;
use Spiral\Boot\Injector\ProvideFrom;

#[ProvideFrom('detect')]
enum DriverEnum implements InjectableEnumInterface
{
    case Database;

    public static function detect(EnvironmentInterface $env): self
    {
        return match ($env->get('PERSISTENCE_DRIVER', 'db')) {
            'cycle', 'database', 'db' => self::Database,
        };
    }
}
