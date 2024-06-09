<?php

declare(strict_types=1);

namespace Modules\Profiler\Application\TopFunctions;

enum Metric: string
{
    case CPU = 'cpu';
    case WallTime = 'wt';
    case MemoryChange = 'pmu';
    case Memory = 'mu';
    case Calls = 'ct';

    case ExclusiveCpu = 'excl_cpu';
    case ExclusiveWallTime = 'excl_wt';
    case ExclusiveMemoryChange = 'excl_pmu';
    case ExclusiveMemory = 'excl_mu';
    case ExclusiveCalls = 'excl_ct';
}
