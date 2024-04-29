<?php

declare(strict_types=1);

namespace App\Application;

use Spiral\Boot\EnvironmentInterface;
use Spiral\Boot\Injector\InjectableEnumInterface;
use Spiral\Boot\Injector\ProvideFrom;

#[ProvideFrom(method: 'detect')]
enum Mode implements InjectableEnumInterface
{
    case Cli;
    case RoadRunner;

    public function insideRoadRunner(): bool
    {
        return $this === self::RoadRunner;
    }

    public static function detect(EnvironmentInterface $environment): self
    {
        $value = $environment->get('MODE');

        return match ($value) {
            'roadrunner', 'rr' => self::RoadRunner,
            default => self::Cli,
        };
    }
}
