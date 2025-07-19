<?php

declare(strict_types=1);

namespace Modules\Profiler\Domain\Edge;

use Cycle\Annotated\Annotation\Column;
use Cycle\Annotated\Annotation\Embeddable;

#[Embeddable(
    columnPrefix: 'd_',
)]
class Diff
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
     * Get diff metric value by name, with safe fallback to 0
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
     * Get all diff metrics as associative array
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
     * Calculate diff from two Cost objects
     */
    public static function fromCosts(Cost $parent, Cost $current): self
    {
        return new self(
            cpu: $parent->cpu - $current->cpu,
            wt: $parent->wt - $current->wt,
            ct: $parent->ct - $current->ct,
            mu: $parent->mu - $current->mu,
            pmu: $parent->pmu - $current->pmu,
        );
    }
}
