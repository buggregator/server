<?php

declare(strict_types=1);

namespace Modules\Profiler\Application\Handlers;

use Modules\Profiler\Application\EventHandlerInterface;
use Modules\Profiler\Application\MetricsHelper;

// TODO: fix diff calculation
final class CalculateDiffsBetweenEdges implements EventHandlerInterface
{
    public function handle(array $event): array
    {
        $data = \array_reverse($event['profile'] ?? []);
        $parents = [];

        foreach ($data as $name => $values) {
            [$parent, $func] = $this->splitName($name);

            if ($parent) {
                $parentValues = $parents[$parent] ?? MetricsHelper::getAllMetrics([]);

                // Use MetricsHelper to safely access metrics with defaults
                $currentMetrics = MetricsHelper::getAllMetrics($values);

                $event['profile'][$name] = \array_merge([
                    'd_cpu' => $parentValues['cpu'] - $currentMetrics['cpu'],
                    'd_wt' => $parentValues['wt'] - $currentMetrics['wt'],
                    'd_mu' => $parentValues['mu'] - $currentMetrics['mu'],
                    'd_pmu' => $parentValues['pmu'] - $currentMetrics['pmu'],
                ], $values);
            }

            // Store normalized metrics for parent lookup
            $parents[$func] = MetricsHelper::getAllMetrics($values);
        }

        return $event;
    }

    /**
     * @return array{0: string|null, 1: string}
     */
    private function splitName(string $name): array
    {
        $a = \explode('==>', $name);
        if (isset($a[1])) {
            return $a;
        }

        return [null, $a[0]];
    }
}
