<?php

declare(strict_types=1);

namespace Tests\Unit\Modules\Profiler\Domain\Edge;

use Modules\Profiler\Domain\Edge\Cost;
use Modules\Profiler\Domain\Edge\Diff;
use PHPUnit\Framework\TestCase;

final class DiffTest extends TestCase
{
    public function testGetMetricReturnsCorrectValues(): void
    {
        $diff = new Diff(cpu: -50, wt: 100, ct: 0, mu: -200, pmu: 500);

        $this->assertSame(-50, $diff->getMetric('cpu'));
        $this->assertSame(100, $diff->getMetric('wt'));
        $this->assertSame(0, $diff->getMetric('ct'));
        $this->assertSame(-200, $diff->getMetric('mu'));
        $this->assertSame(500, $diff->getMetric('pmu'));
    }

    public function testGetMetricReturnsZeroForUnknownMetric(): void
    {
        $diff = new Diff(cpu: -50, wt: 100, ct: 0, mu: -200, pmu: 500);

        $this->assertSame(0, $diff->getMetric('unknown'));
    }

    public function testToArrayReturnsAllMetrics(): void
    {
        $diff = new Diff(cpu: -50, wt: 100, ct: 0, mu: -200, pmu: 500);

        $expected = [
            'cpu' => -50,
            'wt' => 100,
            'ct' => 0,
            'mu' => -200,
            'pmu' => 500,
        ];

        $this->assertSame($expected, $diff->toArray());
    }

    public function testFromCostsCalculatesDiffCorrectly(): void
    {
        $parent = new Cost(cpu: 200, wt: 500, ct: 10, mu: 2048, pmu: 4096);
        $current = new Cost(cpu: 150, wt: 300, ct: 7, mu: 1024, pmu: 3000);

        $diff = Diff::fromCosts($parent, $current);

        $this->assertSame(50, $diff->cpu);     // 200 - 150
        $this->assertSame(200, $diff->wt);     // 500 - 300
        $this->assertSame(3, $diff->ct);       // 10 - 7
        $this->assertSame(1024, $diff->mu);    // 2048 - 1024
        $this->assertSame(1096, $diff->pmu);   // 4096 - 3000
    }

    public function testFromCostsWithNegativeDiff(): void
    {
        $parent = new Cost(cpu: 100, wt: 200, ct: 3, mu: 1000, pmu: 2000);
        $current = new Cost(cpu: 150, wt: 300, ct: 5, mu: 1500, pmu: 3000);

        $diff = Diff::fromCosts($parent, $current);

        $this->assertSame(-50, $diff->cpu);    // 100 - 150
        $this->assertSame(-100, $diff->wt);    // 200 - 300
        $this->assertSame(-2, $diff->ct);      // 3 - 5
        $this->assertSame(-500, $diff->mu);    // 1000 - 1500
        $this->assertSame(-1000, $diff->pmu);  // 2000 - 3000
    }

    public function testFromCostsWithZeroParent(): void
    {
        $parent = new Cost(cpu: 0, wt: 0, ct: 0, mu: 0, pmu: 0);
        $current = new Cost(cpu: 150, wt: 300, ct: 5, mu: 1500, pmu: 3000);

        $diff = Diff::fromCosts($parent, $current);

        $this->assertSame(-150, $diff->cpu);
        $this->assertSame(-300, $diff->wt);
        $this->assertSame(-5, $diff->ct);
        $this->assertSame(-1500, $diff->mu);
        $this->assertSame(-3000, $diff->pmu);
    }

    public function testFromCostsWithZeroCurrent(): void
    {
        $parent = new Cost(cpu: 150, wt: 300, ct: 5, mu: 1500, pmu: 3000);
        $current = new Cost(cpu: 0, wt: 0, ct: 0, mu: 0, pmu: 0);

        $diff = Diff::fromCosts($parent, $current);

        $this->assertSame(150, $diff->cpu);
        $this->assertSame(300, $diff->wt);
        $this->assertSame(5, $diff->ct);
        $this->assertSame(1500, $diff->mu);
        $this->assertSame(3000, $diff->pmu);
    }

    public function testFromCostsWithEqualCosts(): void
    {
        $parent = new Cost(cpu: 150, wt: 300, ct: 5, mu: 1500, pmu: 3000);
        $current = new Cost(cpu: 150, wt: 300, ct: 5, mu: 1500, pmu: 3000);

        $diff = Diff::fromCosts($parent, $current);

        $this->assertSame(0, $diff->cpu);
        $this->assertSame(0, $diff->wt);
        $this->assertSame(0, $diff->ct);
        $this->assertSame(0, $diff->mu);
        $this->assertSame(0, $diff->pmu);
    }
}
