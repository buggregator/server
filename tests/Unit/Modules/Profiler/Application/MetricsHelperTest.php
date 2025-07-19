<?php

declare(strict_types=1);

namespace Tests\Unit\Modules\Profiler\Application;

use Modules\Profiler\Application\MetricsHelper;
use PHPUnit\Framework\TestCase;

final class MetricsHelperTest extends TestCase
{
    public function testGetMetricWithExistingValue(): void
    {
        $data = ['cpu' => 100, 'wt' => 200];

        $this->assertSame(100, MetricsHelper::getMetric($data, 'cpu'));
        $this->assertSame(200, MetricsHelper::getMetric($data, 'wt'));
    }

    public function testGetMetricWithMissingValue(): void
    {
        $data = ['wt' => 200];

        $this->assertSame(0, MetricsHelper::getMetric($data, 'cpu'));
        $this->assertSame(0, MetricsHelper::getMetric($data, 'mu'));
        $this->assertSame(0, MetricsHelper::getMetric($data, 'pmu'));
        $this->assertSame(0, MetricsHelper::getMetric($data, 'ct'));
    }

    public function testNormalizeMetricsWithPartialData(): void
    {
        $input = ['cpu' => 100, 'mu' => 1024];
        $normalized = MetricsHelper::normalizeMetrics($input);

        $this->assertSame([
            'cpu' => 100,
            'wt' => 0,
            'mu' => 1024,
            'pmu' => 0,
            'ct' => 0,
        ], $normalized);
    }

    public function testNormalizeMetricsWithEmptyData(): void
    {
        $normalized = MetricsHelper::normalizeMetrics([]);

        $this->assertSame([
            'cpu' => 0,
            'wt' => 0,
            'mu' => 0,
            'pmu' => 0,
            'ct' => 0,
        ], $normalized);
    }

    public function testGetAllMetricsWithPartialData(): void
    {
        $input = ['wt' => 500, 'ct' => 10];
        $result = MetricsHelper::getAllMetrics($input);

        $this->assertSame([
            'cpu' => 0,
            'wt' => 500,
            'mu' => 0,
            'pmu' => 0,
            'ct' => 10,
        ], $result);
    }

    public function testHasCpuMetricsWithNoCpu(): void
    {
        $this->assertFalse(MetricsHelper::hasCpuMetrics([]));
        $this->assertFalse(MetricsHelper::hasCpuMetrics(['wt' => 100]));
        $this->assertFalse(MetricsHelper::hasCpuMetrics(['cpu' => 0]));
    }

    public function testHasCpuMetricsWithCpu(): void
    {
        $this->assertTrue(MetricsHelper::hasCpuMetrics(['cpu' => 1]));
        $this->assertTrue(MetricsHelper::hasCpuMetrics(['cpu' => 100, 'wt' => 200]));
    }

    public function testGetMetricWithUnknownMetric(): void
    {
        $data = ['custom' => 123];

        // Should return 0 for unknown metrics
        $this->assertSame(0, MetricsHelper::getMetric($data, 'unknown'));
    }
}
