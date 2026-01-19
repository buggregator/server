<?php

declare(strict_types=1);

namespace Modules\Profiler\Domain\Edge;

use Cycle\Annotated\Annotation\Column;
use Cycle\Annotated\Annotation\Embeddable;

#[Embeddable(
    columnPrefix: 'p_',
)]
class Percents
{
    public function __construct(
        #[Column(type: 'float')]
        public float $cpu,
        #[Column(type: 'float')]
        public float $wt,
        #[Column(type: 'float')]
        public float $ct,
        #[Column(type: 'float')]
        public float $mu,
        #[Column(type: 'float')]
        public float $pmu,
    ) {}

    /**
     * Get percentage metric value by name, with safe fallback to 0.0
     */
    public function getMetric(string $metric): float
    {
        return match ($metric) {
            'cpu' => $this->cpu,
            'wt' => $this->wt,
            'ct' => $this->ct,
            'mu' => $this->mu,
            'pmu' => $this->pmu,
            default => 0.0,
        };
    }

    /**
     * Get all percentage metrics as associative array
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
     * Calculate percentages from cost values and totals
     */
    public static function fromCost(Cost $cost, Cost $totals): self
    {
        $calculatePercent = static fn(int $value, int $total): float =>
            $value > 0 && $total > 0 ? round($value / $total * 100, 3) : 0.0;

        return new self(
            cpu: $calculatePercent($cost->cpu, $totals->cpu),
            wt: $calculatePercent($cost->wt, $totals->wt),
            ct: $calculatePercent($cost->ct, $totals->ct),
            mu: $calculatePercent($cost->mu, $totals->mu),
            pmu: $calculatePercent($cost->pmu, $totals->pmu),
        );
    }
}
