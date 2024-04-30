<?php

declare(strict_types=1);

namespace Modules\Metrics\Application;

use Spiral\RoadRunner\Metrics\CollectorInterface;
use Spiral\RoadRunner\Metrics\MetricsInterface;

/**
 * @internal
 * For testing purposes only
 */
final class NullDriver implements MetricsInterface
{
    public function add(string $name, float $value, array $labels = []): void {}

    public function sub(string $name, float $value, array $labels = []): void {}

    public function observe(string $name, float $value, array $labels = []): void {}

    public function set(string $name, float $value, array $labels = []): void {}

    public function declare(string $name, CollectorInterface $collector): void {}

    public function unregister(string $name): void {}
}
