<?php

declare(strict_types=1);

namespace Modules\Profiler\Domain\Edge;

use Cycle\Annotated\Annotation\Column;
use Cycle\Annotated\Annotation\Embeddable;

#[Embeddable]
class Cost
{
    public function __construct(
        #[Column(type: 'integer')]
        public int $cpu,
        #[Column(type: 'integer')]
        public int $wt,
        #[Column(type: 'integer')]
        public int $ct,
        #[Column(type: 'integer')]
        public int $mu,
        #[Column(type: 'integer')]
        public int $pmu,
    ) {}

    /**
     * Get metric value by name, with safe fallback to 0
     */
    public function getMetric(string $metric): int
    {
        return match ($metric) {
            'cpu' => $this->cpu,
            'wt' => $this->wt,
            'ct' => $this->ct,
            'mu' => $this->mu,
            'pmu' => $this->pmu,
            default => 0,
        };
    }

    /**
     * Get all metrics as associative array
     */
    public function toArray(): array
    {
        return [
            'cpu' => $this->cpu,
            'wt' => $this->wt,
            'ct' => $this->ct,
            'mu' => $this->mu,
            'pmu' => $this->pmu,
        ];
    }

    /**
     * Add another Cost to this one (for aggregation)
     */
    public function add(Cost $other): Cost
    {
        return new self(
            cpu: $this->cpu + $other->cpu,
            wt: $this->wt + $other->wt,
            ct: $this->ct + $other->ct,
            mu: $this->mu + $other->mu,
            pmu: $this->pmu + $other->pmu,
        );
    }

    /**
     * Subtract another Cost from this one
     */
    public function subtract(Cost $other): Cost
    {
        return new self(
            cpu: max(0, $this->cpu - $other->cpu),
            wt: max(0, $this->wt - $other->wt),
            ct: max(0, $this->ct - $other->ct),
            mu: max(0, $this->mu - $other->mu),
            pmu: max(0, $this->pmu - $other->pmu),
        );
    }

    /**
     * Check if this cost has any CPU metrics
     */
    public function hasCpuMetrics(): bool
    {
        return $this->cpu > 0;
    }

    /**
     * Create a new Cost with only exclusive metrics (all metrics minus given cost)
     */
    public function getExclusive(Cost $inclusive): Cost
    {
        return $this->subtract($inclusive);
    }
}
