<?php

declare(strict_types=1);

namespace Modules\Profiler\Application\Handlers;

use Modules\Profiler\Application\EventHandlerInterface;

final class PreparePeaks implements EventHandlerInterface
{
    public function handle(array $event): array
    {
        // TODO: fix peaks calculation
        // @see \Modules\Profiler\Interfaces\Queries\FindTopFunctionsByUuidHandler:$overallTotals
        $event['peaks'] = $event['profile']['main()'] ?? [
            'wt' => 0,
            'ct' => 0,
            'mu' => 0,
            'pmu' => 0,
            'cpu' => 0,
        ];

        return $event;
    }
}
