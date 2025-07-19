<?php

declare(strict_types=1);

namespace Tests\Unit\Modules\Profiler\Domain\Edge;

use Modules\Profiler\Domain\Edge\Cost;
use Modules\Profiler\Domain\Edge\Percents;
use PHPUnit\Framework\TestCase;

final class PercentsTest extends TestCase
{
    public function testGetMetricReturnsCorrectValues(): void
    {
        $percents = new Percents(cpu: 10.5, wt: 25.3, ct: 5.0, mu: 15.7, pmu: 30.2);

        $this->assertSame(10.5, $percents->getMetric('cpu'));
        $this->assertSame(25.3, $percents->getMetric('wt'));
        $this->assertSame(5.0, $percents->getMetric('ct'));
        $this->assertSame(15.7, $percents->getMetric('mu'));
        $this->assertSame(30.2, $percents->getMetric('pmu'));
    }

    public function testGetMetricReturnsZeroForUnknownMetric(): void
    {
        $percents = new Percents(cpu: 10.5, wt: 25.3, ct: 5.0, mu: 15.7, pmu: 30.2);

        $this->assertSame(0.0, $percents->getMetric('unknown'));
    }

    public function testToArrayReturnsAllMetrics(): void
    {
        $percents = new Percents(cpu: 10.5, wt: 25.3, ct: 5.0, mu: 15.7, pmu: 30.2);

        $expected = [
            'cpu' => 10.5,
            'wt' => 25.3,
            'ct' => 5.0,
            'mu' => 15.7,
            'pmu' => 30.2,
        ];

        $this->assertSame($expected, $percents->toArray());
    }

    public function testFromCostCalculatesPercentsCorrectly(): void
    {
        $cost = new Cost(cpu: 100, wt: 250, ct: 5, mu: 1024, pmu: 2048);
        $totals = new Cost(cpu: 1000, wt: 1000, ct: 100, mu: 10240, pmu: 10240);

        $percents = Percents::fromCost($cost, $totals);

        $this->assertSame(10.0, $percents->cpu);    // 100/1000 * 100
        $this->assertSame(25.0, $percents->wt);     // 250/1000 * 100
        $this->assertSame(5.0, $percents->ct);      // 5/100 * 100
        $this->assertSame(10.0, $percents->mu);     // 1024/10240 * 100
        $this->assertSame(20.0, $percents->pmu);    // 2048/10240 * 100
    }

    public function testFromCostWithZeroTotalsReturnsZeroPercents(): void
    {
        $cost = new Cost(cpu: 100, wt: 250, ct: 5, mu: 1024, pmu: 2048);
        $totals = new Cost(cpu: 0, wt: 0, ct: 0, mu: 0, pmu: 0);

        $percents = Percents::fromCost($cost, $totals);

        $this->assertSame(0.0, $percents->cpu);
        $this->assertSame(0.0, $percents->wt);
        $this->assertSame(0.0, $percents->ct);
        $this->assertSame(0.0, $percents->mu);
        $this->assertSame(0.0, $percents->pmu);
    }

    public function testFromCostWithZeroCostReturnsZeroPercents(): void
    {
        $cost = new Cost(cpu: 0, wt: 0, ct: 0, mu: 0, pmu: 0);
        $totals = new Cost(cpu: 1000, wt: 1000, ct: 100, mu: 10240, pmu: 10240);

        $percents = Percents::fromCost($cost, $totals);

        $this->assertSame(0.0, $percents->cpu);
        $this->assertSame(0.0, $percents->wt);
        $this->assertSame(0.0, $percents->ct);
        $this->assertSame(0.0, $percents->mu);
        $this->assertSame(0.0, $percents->pmu);
    }

    public function testFromCostRoundsToThreeDecimals(): void
    {
        $cost = new Cost(cpu: 333, wt: 666, ct: 7, mu: 3333, pmu: 6666);
        $totals = new Cost(cpu: 1000, wt: 2000, ct: 20, mu: 10000, pmu: 20000);

        $percents = Percents::fromCost($cost, $totals);

        $this->assertSame(33.3, $percents->cpu);    // 333/1000 * 100 = 33.3
        $this->assertSame(33.3, $percents->wt);     // 666/2000 * 100 = 33.3
        $this->assertSame(35.0, $percents->ct);     // 7/20 * 100 = 35.0
        $this->assertSame(33.33, $percents->mu);    // 3333/10000 * 100 = 33.33
        $this->assertSame(33.33, $percents->pmu);   // 6666/20000 * 100 = 33.33
    }

    public function testFromCostWithMixedZeroValues(): void
    {
        $cost = new Cost(cpu: 100, wt: 0, ct: 5, mu: 0, pmu: 2048);
        $totals = new Cost(cpu: 1000, wt: 2000, ct: 0, mu: 10240, pmu: 0);

        $percents = Percents::fromCost($cost, $totals);

        $this->assertSame(10.0, $percents->cpu);  // Normal calculation
        $this->assertSame(0.0, $percents->wt);    // Zero cost
        $this->assertSame(0.0, $percents->ct);    // Zero total
        $this->assertSame(0.0, $percents->mu);    // Zero cost
        $this->assertSame(0.0, $percents->pmu);   // Zero total
    }
}
