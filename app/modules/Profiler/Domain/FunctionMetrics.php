<?php

declare(strict_types=1);

namespace Modules\Profiler\Domain;

use Modules\Profiler\Domain\Edge\Cost;

/**
 * Represents aggregated metrics for a single function across all its calls
 */
final readonly class FunctionMetrics
{
    public function __construct(
        public string $function,
        public Cost $inclusive,
        public Cost $exclusive,
    ) {}

    /**
     * Create initial function metrics from first edge
     */
    public static function fromEdge(Edge $edge): self
    {
        return new self(
            function: $edge->getCallee(),
            inclusive: $edge->getCost(),
            exclusive: $edge->getCost(), // Initially same as inclusive
        );
    }

    /**
     * Add metrics from another edge call to the same function
     */
    public function addEdge(Edge $edge): self
    {
        return new self(
            function: $this->function,
            inclusive: $this->inclusive->add($edge->getCost()),
            exclusive: $this->exclusive->add($edge->getCost()),
        );
    }

    /**
     * Subtract child function costs from exclusive metrics
     */
    public function subtractChild(Cost $childCost): self
    {
        return new self(
            function: $this->function,
            inclusive: $this->inclusive,
            exclusive: $this->exclusive->subtract($childCost),
        );
    }

    /**
     * Get metric value for sorting
     */
    public function getMetricForSort(string $metric): int
    {
        // Handle exclusive metrics
        if (str_starts_with($metric, 'excl_')) {
            $baseMetric = substr($metric, 5);
            return $this->exclusive->getMetric($baseMetric);
        }

        return $this->inclusive->getMetric($metric);
    }

    /**
     * Convert to array format expected by the frontend
     */
    public function toArray(Cost $overallTotals): array
    {
        $result = [
            'function' => $this->function,
            ...$this->inclusive->toArray(),
        ];

        // Add exclusive metrics
        foreach (['cpu', 'wt', 'mu', 'pmu', 'ct'] as $metric) {
            $result['excl_' . $metric] = $this->exclusive->getMetric($metric);
        }

        // Calculate percentages
        foreach (['cpu', 'wt', 'mu', 'pmu', 'ct'] as $metric) {
            $totalValue = $overallTotals->getMetric($metric);

            $result['p_' . $metric] = $this->calculatePercentage(
                $this->inclusive->getMetric($metric),
                $totalValue,
            );

            $result['p_excl_' . $metric] = $this->calculatePercentage(
                $this->exclusive->getMetric($metric),
                $totalValue,
            );
        }

        return $result;
    }

    private function calculatePercentage(int $value, int $total): float
    {
        return $value > 0 && $total > 0
            ? round($value / $total * 100, 3)
            : 0.0;
    }
}
