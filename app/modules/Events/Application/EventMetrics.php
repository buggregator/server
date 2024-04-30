<?php

declare(strict_types=1);

namespace Modules\Events\Application;

use Spiral\RoadRunner\Metrics\MetricsInterface;

final readonly class EventMetrics
{
    public function __construct(
        private MetricsInterface $metrics,
    ) {
    }

    public function newEvent(string $type): void
    {
        $this->metrics->add('events', 1, [$type]);
    }
}
