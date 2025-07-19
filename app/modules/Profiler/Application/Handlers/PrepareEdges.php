<?php

declare(strict_types=1);

namespace Modules\Profiler\Application\Handlers;

use Modules\Profiler\Application\EventHandlerInterface;
use Modules\Profiler\Application\MetricsHelper;

final class PrepareEdges implements EventHandlerInterface
{
    public function handle(array $event): array
    {
        $data = \array_reverse($event['profile'] ?? []);

        $event['edges'] = [];
        $parents = [];

        $prev = null;

        $id = 1;
        foreach ($data as $name => $values) {
            [$parent, $func] = $this->splitName($name);

            $parentId = $parents[$parent] ?? $prev;

            // Normalize metrics to ensure all required fields are present
            $normalizedValues = MetricsHelper::getAllMetrics($values);

            // Calculate percentages with safe metric access
            foreach (['cpu', 'mu', 'pmu', 'wt'] as $key) {
                $peakValue = MetricsHelper::getMetric($event['peaks'], $key);
                $values['p_' . $key] = \round(
                    $normalizedValues[$key] > 0 && $peakValue > 0
                        ? ($normalizedValues[$key]) / $peakValue * 100
                        : 0,
                    3,
                );
            }

            $event['edges']['e' . $id] = [
                'id' => 'e' . $id,
                'caller' => $parent,
                'callee' => $func,
                'cost' => $values,
                'parent' => $parentId,
            ];

            $parents[$func] = 'e' . $id;
            $prev = 'e' . $id;

            $id++;
        }

        $event['total_edges'] = \count($event['edges']);

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
