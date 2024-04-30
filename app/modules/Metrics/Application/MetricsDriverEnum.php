<?php

declare(strict_types=1);

namespace Modules\Metrics\Application;

use Spiral\Boot\EnvironmentInterface;
use Spiral\Boot\Injector\InjectableEnumInterface;
use Spiral\Boot\Injector\ProvideFrom;

#[ProvideFrom('detect')]
enum MetricsDriverEnum implements InjectableEnumInterface
{
    case Null;
    case RoadRunner;

    public static function detect(EnvironmentInterface $env): self
    {
        return match ($env->get('METRICS_DRIVER', 'null')) {
            'rr', 'roadrunner' => self::RoadRunner,
            default => self::Null,
        };
    }
}
