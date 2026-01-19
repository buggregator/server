<?php

declare(strict_types=1);

namespace Modules\Profiler\Application\Handlers;

use Modules\Profiler\Application\EventHandlerInterface;
use Modules\Profiler\Application\MetricsHelper;

final class PreparePeaks implements EventHandlerInterface
{
    public function handle(array $event): array
    {
        // TODO: fix peaks calculation
        // @see \Modules\Profiler\Interfaces\Queries\FindTopFunctionsByUuidHandler:$overallTotals

        // Get main() metrics or use defaults if not available
        $mainMetrics = $event['profile']['main()'] ?? [];

        $event['peaks'] = MetricsHelper::getAllMetrics($mainMetrics);

        return $event;
    }
}
