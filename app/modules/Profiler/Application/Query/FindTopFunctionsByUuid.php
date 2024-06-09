<?php

declare(strict_types=1);

namespace Modules\Profiler\Application\Query;

use App\Application\Domain\ValueObjects\Uuid;
use Modules\Profiler\Application\TopFunctions\Metric;
use Spiral\Cqrs\QueryInterface;

final class FindTopFunctionsByUuid implements QueryInterface
{
    public function __construct(
        public Uuid $profileUuid,
        public int $limit = 100,
        public Metric $metric = Metric::CPU,
    ) {}
}
