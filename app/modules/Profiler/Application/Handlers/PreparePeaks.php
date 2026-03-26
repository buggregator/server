<?php

declare(strict_types=1);

namespace Modules\Profiler\Application\Handlers;

use Modules\Profiler\Application\EventHandlerInterface;

final class PreparePeaks implements EventHandlerInterface
{
    private const METRIC_DEFAULTS = [
        'wt' => 0,
        'ct' => 0,
        'mu' => 0,
        'pmu' => 0,
        'cpu' => 0,
    ];

    public function handle(array $event): array
    {
        // Normalize: fill missing metrics with 0 for every profile entry
        // (cpu is absent when XHProf runs without XHPROF_FLAGS_CPU)
        foreach ($event['profile'] as $name => $values) {
            $event['profile'][$name] = $values + self::METRIC_DEFAULTS;
        }

        // TODO: fix peaks calculation
        // @see \Modules\Profiler\Interfaces\Queries\FindTopFunctionsByUuidHandler:$overallTotals
        $event['peaks'] = $event['profile']['main()'] ?? self::METRIC_DEFAULTS;

        return $event;
    }
}
