<?php

declare(strict_types=1);

namespace Modules\Profiler\Application;

/**
 * Helper class for safely handling profiler metrics with potential missing values
 */
final class MetricsHelper
{
    /**
     * Default metric values when not available in profile data
     */
    private const DEFAULT_METRICS = [
        'cpu' => 0,
        'wt' => 0,
        'mu' => 0,
        'pmu' => 0,
        'ct' => 0,
    ];

    /**
     * Get metric value with fallback to default if missing
     */
    public static function getMetric(array $data, string $metric): int|float
    {
        return $data[$metric] ?? self::DEFAULT_METRICS[$metric] ?? 0;
    }

    /**
     * Normalize metrics array by ensuring all required metrics are present
     */
    public static function normalizeMetrics(array $metrics): array
    {
        return \array_merge(self::DEFAULT_METRICS, $metrics);
    }

    /**
     * Get all available metrics from data, with defaults for missing ones
     */
    public static function getAllMetrics(array $data): array
    {
        return [
            'cpu' => self::getMetric($data, 'cpu'),
            'wt' => self::getMetric($data, 'wt'),
            'mu' => self::getMetric($data, 'mu'),
            'pmu' => self::getMetric($data, 'pmu'),
            'ct' => self::getMetric($data, 'ct'),
        ];
    }

    /**
     * Check if any CPU-related metrics are available
     */
    public static function hasCpuMetrics(array $data): bool
    {
        return isset($data['cpu']) && $data['cpu'] > 0;
    }
}
