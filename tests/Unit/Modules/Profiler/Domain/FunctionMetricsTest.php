<?php

declare(strict_types=1);

namespace Tests\Unit\Modules\Profiler\Domain;

use Modules\Profiler\Domain\Edge;
use Modules\Profiler\Domain\Edge\Cost;
use Modules\Profiler\Domain\Edge\Diff;
use Modules\Profiler\Domain\Edge\Percents;
use Modules\Profiler\Domain\FunctionMetrics;
use App\Application\Domain\ValueObjects\Uuid;
use PHPUnit\Framework\TestCase;

final class FunctionMetricsTest extends TestCase
{
    public function testFromEdge(): void
    {
        $edge = $this->createEdge('testFunction', 100, 200);
        $metrics = FunctionMetrics::fromEdge($edge);

        $this->assertSame('testFunction', $metrics->function);
        $this->assertSame(100, $metrics->inclusive->cpu);
        $this->assertSame(200, $metrics->inclusive->wt);
        // Initially, exclusive equals inclusive
        $this->assertSame(100, $metrics->exclusive->cpu);
        $this->assertSame(200, $metrics->exclusive->wt);
    }

    public function testAddEdge(): void
    {
        $edge1 = $this->createEdge('testFunction', 100, 200);
        $edge2 = $this->createEdge('testFunction', 50, 100);

        $metrics = FunctionMetrics::fromEdge($edge1);
        $updated = $metrics->addEdge($edge2);

        $this->assertSame('testFunction', $updated->function);
        $this->assertSame(150, $updated->inclusive->cpu);
        $this->assertSame(300, $updated->inclusive->wt);
    }

    public function testSubtractChild(): void
    {
        $edge = $this->createEdge('parentFunction', 100, 200);
        $metrics = FunctionMetrics::fromEdge($edge);

        $childCost = new Cost(cpu: 30, wt: 50, ct: 1, mu: 100, pmu: 200);
        $updated = $metrics->subtractChild($childCost);

        // Inclusive should remain the same
        $this->assertSame(100, $updated->inclusive->cpu);
        $this->assertSame(200, $updated->inclusive->wt);

        // Exclusive should be reduced by child cost
        $this->assertSame(70, $updated->exclusive->cpu);
        $this->assertSame(150, $updated->exclusive->wt);
    }

    public function testGetMetricForSortInclusive(): void
    {
        $edge = $this->createEdge('testFunction', 100, 200);
        $metrics = FunctionMetrics::fromEdge($edge);

        $this->assertSame(100, $metrics->getMetricForSort('cpu'));
        $this->assertSame(200, $metrics->getMetricForSort('wt'));
    }

    public function testGetMetricForSortExclusive(): void
    {
        $edge = $this->createEdge('testFunction', 100, 200);
        $metrics = FunctionMetrics::fromEdge($edge);

        $childCost = new Cost(cpu: 30, wt: 50, ct: 1, mu: 100, pmu: 200);
        $updated = $metrics->subtractChild($childCost);

        $this->assertSame(70, $updated->getMetricForSort('excl_cpu'));
        $this->assertSame(150, $updated->getMetricForSort('excl_wt'));
    }

    public function testToArray(): void
    {
        $edge = $this->createEdge('testFunction', 100, 200, 2, 1024, 2048);
        $metrics = FunctionMetrics::fromEdge($edge);

        $childCost = new Cost(cpu: 30, wt: 50, ct: 1, mu: 200, pmu: 400);
        $updated = $metrics->subtractChild($childCost);

        $overallTotals = new Cost(cpu: 1000, wt: 2000, ct: 20, mu: 10240, pmu: 20480);
        $result = $updated->toArray($overallTotals);

        $this->assertSame('testFunction', $result['function']);

        // Inclusive metrics
        $this->assertSame(100, $result['cpu']);
        $this->assertSame(200, $result['wt']);
        $this->assertSame(2, $result['ct']);
        $this->assertSame(1024, $result['mu']);
        $this->assertSame(2048, $result['pmu']);

        // Exclusive metrics
        $this->assertSame(70, $result['excl_cpu']);
        $this->assertSame(150, $result['excl_wt']);
        $this->assertSame(1, $result['excl_ct']);
        $this->assertSame(824, $result['excl_mu']);
        $this->assertSame(1648, $result['excl_pmu']);

        // Percentages (10% and 7% respectively)
        $this->assertSame(10.0, $result['p_cpu']);
        $this->assertSame(10.0, $result['p_wt']);
        $this->assertSame(7.0, $result['p_excl_cpu']);
        $this->assertSame(7.5, $result['p_excl_wt']);
    }

    private function createEdge(
        string $callee,
        int $cpu = 0,
        int $wt = 0,
        int $ct = 1,
        int $mu = 0,
        int $pmu = 0,
    ): Edge {
        return new Edge(
            uuid: Uuid::generate(),
            profileUuid: Uuid::generate(),
            order: 1,
            cost: new Cost(cpu: $cpu, wt: $wt, ct: $ct, mu: $mu, pmu: $pmu),
            diff: new Diff(cpu: 0, wt: 0, ct: 0, mu: 0, pmu: 0),
            percents: new Percents(cpu: 0.0, wt: 0.0, ct: 0.0, mu: 0.0, pmu: 0.0),
            callee: $callee,
        );
    }
}
