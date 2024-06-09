<?php

declare(strict_types=1);

namespace Modules\Profiler\Application\Query;

use App\Application\Domain\ValueObjects\Uuid;
use Modules\Profiler\Application\CallGraph\Metric;
use Spiral\Cqrs\QueryInterface;

final readonly class FindCallGraphByUuid implements QueryInterface
{
    public function __construct(
        public Uuid $profileUuid,
        public int|float $threshold = 0,
        public int $percentage = 0,
        public Metric $metric = Metric::CPU,
    ) {}
}
