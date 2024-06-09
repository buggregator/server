<?php

declare(strict_types=1);

namespace Modules\Profiler\Application\CallGraph;

enum Metric: string
{
    case CPU = 'cpu';
    case WallTime = 'wt';
    case MemoryChange = 'pmu';
    case Memory = 'mu';
    case Calls = 'ct';
}
