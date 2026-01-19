<?php

declare(strict_types=1);

namespace Tests\Unit\Modules\Profiler\Domain\Edge;

use Modules\Profiler\Domain\Edge\Cost;
use PHPUnit\Framework\TestCase;

final class CostTest extends TestCase
{
    public function testGetMetricReturnsCorrectValues(): void
    {
        $cost = new Cost(cpu: 100, wt: 200, ct: 5, mu: 1024, pmu: 2048);

        $this->assertSame(100, $cost->getMetric('cpu'));
        $this->assertSame(200, $cost->getMetric('wt'));
        $this->assertSame(5, $cost->getMetric('ct'));
        $this->assertSame(1024, $cost->getMetric('mu'));
        $this->assertSame(2048, $cost->getMetric('pmu'));
    }

    public function testGetMetricReturnsZeroForUnknownMetric(): void
    {
        $cost = new Cost(cpu: 100, wt: 200, ct: 5, mu: 1024, pmu: 2048);

        $this->assertSame(0, $cost->getMetric('unknown'));
    }

    public function testToArrayReturnsAllMetrics(): void
    {
        $cost = new Cost(cpu: 100, wt: 200, ct: 5, mu: 1024, pmu: 2048);

        $expected = [
            'cpu' => 100,
            'wt' => 200,
            'ct' => 5,
            'mu' => 1024,
            'pmu' => 2048,
        ];

        $this->assertSame($expected, $cost->toArray());
    }

    public function testAddCombinesCosts(): void
    {
        $cost1 = new Cost(cpu: 100, wt: 200, ct: 2, mu: 1024, pmu: 2048);
        $cost2 = new Cost(cpu: 50, wt: 100, ct: 3, mu: 512, pmu: 1024);

        $result = $cost1->add($cost2);

        $this->assertSame(150, $result->cpu);
        $this->assertSame(300, $result->wt);
        $this->assertSame(5, $result->ct);
        $this->assertSame(1536, $result->mu);
        $this->assertSame(3072, $result->pmu);
    }

    public function testSubtractWithPositiveResults(): void
    {
        $cost1 = new Cost(cpu: 100, wt: 200, ct: 5, mu: 1024, pmu: 2048);
        $cost2 = new Cost(cpu: 30, wt: 50, ct: 2, mu: 200, pmu: 500);

        $result = $cost1->subtract($cost2);

        $this->assertSame(70, $result->cpu);
        $this->assertSame(150, $result->wt);
        $this->assertSame(3, $result->ct);
        $this->assertSame(824, $result->mu);
        $this->assertSame(1548, $result->pmu);
    }

    public function testSubtractPreventsNegativeValues(): void
    {
        $cost1 = new Cost(cpu: 50, wt: 100, ct: 2, mu: 500, pmu: 1000);
        $cost2 = new Cost(cpu: 100, wt: 200, ct: 5, mu: 1000, pmu: 2000);

        $result = $cost1->subtract($cost2);

        // All values should be 0 (not negative)
        $this->assertSame(0, $result->cpu);
        $this->assertSame(0, $result->wt);
        $this->assertSame(0, $result->ct);
        $this->assertSame(0, $result->mu);
        $this->assertSame(0, $result->pmu);
    }

    public function testHasCpuMetricsWithCpu(): void
    {
        $cost = new Cost(cpu: 100, wt: 200, ct: 5, mu: 1024, pmu: 2048);
        $this->assertTrue($cost->hasCpuMetrics());
    }

    public function testHasCpuMetricsWithoutCpu(): void
    {
        $cost = new Cost(cpu: 0, wt: 200, ct: 5, mu: 1024, pmu: 2048);
        $this->assertFalse($cost->hasCpuMetrics());
    }

    public function testGetExclusive(): void
    {
        $inclusive = new Cost(cpu: 100, wt: 200, ct: 5, mu: 1024, pmu: 2048);
        $child = new Cost(cpu: 30, wt: 50, ct: 1, mu: 200, pmu: 400);

        $exclusive = $inclusive->getExclusive($child);

        $this->assertSame(70, $exclusive->cpu);
        $this->assertSame(150, $exclusive->wt);
        $this->assertSame(4, $exclusive->ct);
        $this->assertSame(824, $exclusive->mu);
        $this->assertSame(1648, $exclusive->pmu);
    }
}
