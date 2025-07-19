<?php

declare(strict_types=1);

namespace Modules\Profiler\Application\Service;

use Modules\Profiler\Domain\Edge;
use Modules\Profiler\Domain\Edge\Cost;
use Modules\Profiler\Domain\FunctionMetrics;

/**
 * Service for aggregating and calculating function performance metrics
 */
final class FunctionMetricsCalculator
{
    /**
     * Calculate aggregated function metrics from edges
     *
     * @param Edge[] $edges
     * @return FunctionMetrics[]
     */
    public function calculateMetrics(array $edges): array
    {
        $functions = [];
        $overallTotals = $this->initializeOverallTotals();

        // First pass: aggregate inclusive metrics per function
        foreach ($edges as $edge) {
            $functionName = $edge->getCallee();

            if (!isset($functions[$functionName])) {
                $functions[$functionName] = FunctionMetrics::fromEdge($edge);
            } else {
                $functions[$functionName] = $functions[$functionName]->addEdge($edge);
            }
        }

        // Calculate overall totals from main() function or first function
        $overallTotals = $this->calculateOverallTotals($functions);

        // Second pass: calculate exclusive metrics by subtracting child costs
        $functions = $this->calculateExclusiveMetrics($functions, $edges);

        return [$functions, $overallTotals];
    }

    /**
     * Sort functions by specified metric
     *
     * @param FunctionMetrics[] $functions
     */
    public function sortFunctions(array $functions, string $sortMetric): array
    {
        usort(
            $functions,
            static fn(FunctionMetrics $a, FunctionMetrics $b) => $b->getMetricForSort(
                $sortMetric,
            ) <=> $a->getMetricForSort($sortMetric),
        );

        return $functions;
    }

    /**
     * Convert function metrics to array format for API response
     *
     * @param FunctionMetrics[] $functions
     */
    public function toArrayFormat(array $functions, Cost $overallTotals): array
    {
        return array_map(
            fn(FunctionMetrics $metrics) => $metrics->toArray($overallTotals),
            $functions,
        );
    }

    private function initializeOverallTotals(): Cost
    {
        return new Cost(cpu: 0, wt: 0, ct: 0, mu: 0, pmu: 0);
    }

    /**
     * @param FunctionMetrics[] $functions
     */
    private function calculateOverallTotals(array $functions): Cost
    {
        // Try to get totals from main() function first
        if (isset($functions['main()'])) {
            return $functions['main()']->inclusive;
        }

        // If no main(), calculate from all functions (less accurate but workable)
        $totals = $this->initializeOverallTotals();

        foreach ($functions as $function) {
            // Only add call counts, other metrics should not be summed across functions
            $totals = new Cost(
                cpu: max($totals->cpu, $function->inclusive->cpu),
                wt: max($totals->wt, $function->inclusive->wt),
                ct: $totals->ct + $function->inclusive->ct,
                mu: max($totals->mu, $function->inclusive->mu),
                pmu: max($totals->pmu, $function->inclusive->pmu),
            );
        }

        return $totals;
    }

    /**
     * Calculate exclusive metrics by subtracting child function costs
     *
     * @param FunctionMetrics[] $functions
     * @param Edge[] $edges
     * @return FunctionMetrics[]
     */
    private function calculateExclusiveMetrics(array $functions, array $edges): array
    {
        // Build parent-child relationships and subtract child costs
        foreach ($edges as $edge) {
            $caller = $edge->getCaller();

            if ($caller && isset($functions[$caller])) {
                $functions[$caller] = $functions[$caller]->subtractChild($edge->getCost());
            }
        }

        return $functions;
    }
}
