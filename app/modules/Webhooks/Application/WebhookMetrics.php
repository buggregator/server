<?php

declare(strict_types=1);

namespace Modules\Webhooks\Application;

use Spiral\RoadRunner\Metrics\MetricsInterface;

final readonly class WebhookMetrics
{
    public function __construct(
        private MetricsInterface $metrics,
    ) {}

    public function called(string $event, string $url, bool $success): void
    {
        $this->metrics->add('webhooks', 1, [$event, $url, $success ? 'true' : 'false']);
    }
}
