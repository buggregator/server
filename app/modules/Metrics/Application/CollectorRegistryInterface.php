<?php

declare(strict_types=1);

namespace Modules\Metrics\Application;

use Spiral\RoadRunner\Metrics\CollectorInterface;

interface CollectorRegistryInterface
{
    public function register(string $name, CollectorInterface $collector): void;
}
