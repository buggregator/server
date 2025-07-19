<?php

declare(strict_types=1);

namespace Tests\Integration\Modules\Profiler;

use Modules\Profiler\Application\Handlers\CalculateDiffsBetweenEdges;
use Modules\Profiler\Application\Handlers\PrepareEdges;
use Modules\Profiler\Application\Handlers\PreparePeaks;
use PHPUnit\Framework\TestCase;

/**
 * Integration test for the profiler event processing pipeline
 */
final class ProfilerEventProcessingTest extends TestCase
{
    public function testCompleteEventProcessingPipeline(): void
    {
        // Simulate a complete profiling event with missing CPU metrics
        $originalEvent = [
            'profile' => [
                'main()' => ['wt' => 5000, 'mu' => 50000, 'pmu' => 60000, 'ct' => 1],
                'main()==>App\\Controller::index' => ['wt' => 3000, 'mu' => 30000, 'pmu' => 35000, 'ct' => 1],
                'App\\Controller::index==>App\\Service::process' => ['wt' => 2000, 'mu' => 20000, 'pmu' => 25000, 'ct' => 1],
                'App\\Service::process==>strlen' => ['wt' => 100, 'mu' => 1000, 'pmu' => 1200, 'ct' => 10],
            ],
            'app_name' => 'Test Application',
            'hostname' => 'test-host',
            'tags' => ['env' => 'test', 'version' => '1.0'],
            'date' => 1714289301,
        ];

        // Step 1: PreparePeaks
        $preparePeaks = new PreparePeaks();
        $event = $preparePeaks->handle($originalEvent);

        $this->assertArrayHasKey('peaks', $event);
        $peaks = $event['peaks'];
        $this->assertSame(0, $peaks['cpu']); // CPU should be 0 (missing)
        $this->assertSame(5000, $peaks['wt']); // From main()
        $this->assertSame(50000, $peaks['mu']); // From main()
        $this->assertSame(60000, $peaks['pmu']); // From main()
        $this->assertSame(1, $peaks['ct']); // From main()

        // Step 2: CalculateDiffsBetweenEdges
        $calculateDiffs = new CalculateDiffsBetweenEdges();
        $event = $calculateDiffs->handle($event);

        // Check that diffs were calculated safely despite missing CPU
        $controllerCall = $event['profile']['main()==>App\\Controller::index'];
        $this->assertSame(0, $controllerCall['d_cpu']); // 0 - 0 = 0
        $this->assertSame(2000, $controllerCall['d_wt']); // 5000 - 3000 = 2000
        $this->assertSame(20000, $controllerCall['d_mu']); // 50000 - 30000 = 20000

        $serviceCall = $event['profile']['App\\Controller::index==>App\\Service::process'];
        $this->assertSame(0, $serviceCall['d_cpu']); // 0 - 0 = 0
        $this->assertSame(1000, $serviceCall['d_wt']); // 3000 - 2000 = 1000

        // Step 3: PrepareEdges
        $prepareEdges = new PrepareEdges();
        $event = $prepareEdges->handle($event);

        $this->assertArrayHasKey('edges', $event);
        $this->assertArrayHasKey('total_edges', $event);
        $this->assertSame(4, $event['total_edges']);

        $edges = $event['edges'];
        $this->assertCount(4, $edges);

        // Check edge structure and percentage calculations
        $strlenEdge = $edges['e1']; // First after reverse
        $this->assertSame('strlen', $strlenEdge['callee']);
        $this->assertSame('App\\Service::process', $strlenEdge['caller']);
        $this->assertSame(0.0, $strlenEdge['cost']['p_cpu']); // 0/0 safely handled
        $this->assertSame(2.0, $strlenEdge['cost']['p_wt']); // 100/5000 * 100

        $serviceEdge = $edges['e2'];
        $this->assertSame('App\\Service::process', $serviceEdge['callee']);
        $this->assertSame('App\\Controller::index', $serviceEdge['caller']);
        $this->assertSame('e3', $serviceEdge['parent']); // Should reference controller edge

        $controllerEdge = $edges['e3'];
        $this->assertSame('App\\Controller::index', $controllerEdge['callee']);
        $this->assertSame('main()', $controllerEdge['caller']);
        $this->assertSame('e4', $controllerEdge['parent']); // Should reference main edge

        $mainEdge = $edges['e4'];
        $this->assertSame('main()', $mainEdge['callee']);
        $this->assertNull($mainEdge['caller']);
        $this->assertNull($mainEdge['parent']);

        // Verify original event data is preserved
        $this->assertSame('Test Application', $event['app_name']);
        $this->assertSame('test-host', $event['hostname']);
        $this->assertSame(['env' => 'test', 'version' => '1.0'], $event['tags']);
        $this->assertSame(1714289301, $event['date']);
    }

    public function testPipelineWithFullMetrics(): void
    {
        $originalEvent = [
            'profile' => [
                'main()' => ['cpu' => 1000, 'wt' => 5000, 'mu' => 50000, 'pmu' => 60000, 'ct' => 1],
                'main()==>processData' => ['cpu' => 600, 'wt' => 3000, 'mu' => 30000, 'pmu' => 35000, 'ct' => 1],
                'processData==>array_filter' => ['cpu' => 200, 'wt' => 1000, 'mu' => 10000, 'pmu' => 12000, 'ct' => 5],
            ],
            'app_name' => 'Full Metrics Test',
            'hostname' => 'production',
            'date' => 1714289302,
        ];

        // Process through pipeline
        $preparePeaks = new PreparePeaks();
        $event = $preparePeaks->handle($originalEvent);

        $calculateDiffs = new CalculateDiffsBetweenEdges();
        $event = $calculateDiffs->handle($event);

        $prepareEdges = new PrepareEdges();
        $event = $prepareEdges->handle($event);

        // Verify peaks calculation with CPU
        $this->assertSame(1000, $event['peaks']['cpu']);
        $this->assertSame(5000, $event['peaks']['wt']);

        // Verify diff calculations with CPU
        $processDataCall = $event['profile']['main()==>processData'];
        $this->assertSame(400, $processDataCall['d_cpu']); // 1000 - 600
        $this->assertSame(2000, $processDataCall['d_wt']); // 5000 - 3000

        $arrayFilterCall = $event['profile']['processData==>array_filter'];
        $this->assertSame(400, $arrayFilterCall['d_cpu']); // 600 - 200
        $this->assertSame(2000, $arrayFilterCall['d_wt']); // 3000 - 1000

        // Verify percentage calculations with CPU
        $edges = $event['edges'];
        $arrayFilterEdge = $edges['e1']; // First after reverse
        $this->assertSame(20.0, $arrayFilterEdge['cost']['p_cpu']); // 200/1000 * 100
        $this->assertSame(20.0, $arrayFilterEdge['cost']['p_wt']); // 1000/5000 * 100

        $processDataEdge = $edges['e2'];
        $this->assertSame(60.0, $processDataEdge['cost']['p_cpu']); // 600/1000 * 100
        $this->assertSame(60.0, $processDataEdge['cost']['p_wt']); // 3000/5000 * 100

        $mainEdge = $edges['e3'];
        $this->assertSame(100.0, $mainEdge['cost']['p_cpu']); // 1000/1000 * 100
        $this->assertSame(100.0, $mainEdge['cost']['p_wt']); // 5000/5000 * 100
    }

    public function testPipelineWithMixedMetrics(): void
    {
        $originalEvent = [
            'profile' => [
                'main()' => ['cpu' => 1000, 'wt' => 5000, 'mu' => 50000, 'pmu' => 60000, 'ct' => 1],
                'main()==>funcA' => ['wt' => 3000, 'mu' => 30000, 'ct' => 1], // Missing cpu, pmu
                'funcA==>funcB' => ['cpu' => 200, 'wt' => 1000, 'pmu' => 12000, 'ct' => 2], // Missing mu
            ],
            'app_name' => 'Mixed Metrics Test',
            'hostname' => 'staging',
            'date' => 1714289303,
        ];

        // Process through pipeline
        $preparePeaks = new PreparePeaks();
        $event = $preparePeaks->handle($originalEvent);

        $calculateDiffs = new CalculateDiffsBetweenEdges();
        $event = $calculateDiffs->handle($event);

        $prepareEdges = new PrepareEdges();
        $event = $prepareEdges->handle($event);

        // Verify mixed metrics handling
        $this->assertSame(1000, $event['peaks']['cpu']); // From main()
        $this->assertSame(5000, $event['peaks']['wt']); // From main()
        $this->assertSame(50000, $event['peaks']['mu']); // From main()
        $this->assertSame(60000, $event['peaks']['pmu']); // From main()

        // Verify diff calculations with mixed metrics
        $funcACall = $event['profile']['main()==>funcA'];
        $this->assertSame(1000, $funcACall['d_cpu']); // 1000 - 0 = 1000
        $this->assertSame(2000, $funcACall['d_wt']); // 5000 - 3000 = 2000
        $this->assertSame(20000, $funcACall['d_mu']); // 50000 - 30000 = 20000
        $this->assertSame(60000, $funcACall['d_pmu']); // 60000 - 0 = 60000

        $funcBCall = $event['profile']['funcA==>funcB'];
        $this->assertSame(-200, $funcBCall['d_cpu']); // 0 - 200 = -200
        $this->assertSame(2000, $funcBCall['d_wt']); // 3000 - 1000 = 2000
        $this->assertSame(30000, $funcBCall['d_mu']); // 30000 - 0 = 30000
        $this->assertSame(-12000, $funcBCall['d_pmu']); // 0 - 12000 = -12000

        // Verify percentage calculations with mixed metrics
        $edges = $event['edges'];

        $funcBEdge = $edges['e1']; // First after reverse
        $this->assertSame(20.0, $funcBEdge['cost']['p_cpu']); // 200/1000 * 100
        $this->assertSame(20.0, $funcBEdge['cost']['p_wt']); // 1000/5000 * 100
        $this->assertSame(0.0, $funcBEdge['cost']['p_mu']); // 0/50000 * 100
        $this->assertSame(20.0, $funcBEdge['cost']['p_pmu']); // 12000/60000 * 100

        $funcAEdge = $edges['e2'];
        $this->assertSame(0.0, $funcAEdge['cost']['p_cpu']); // 0/1000 * 100
        $this->assertSame(60.0, $funcAEdge['cost']['p_wt']); // 3000/5000 * 100
        $this->assertSame(60.0, $funcAEdge['cost']['p_mu']); // 30000/50000 * 100
        $this->assertSame(0.0, $funcAEdge['cost']['p_pmu']); // 0/60000 * 100
    }

    public function testPipelineWithEmptyProfile(): void
    {
        $originalEvent = [
            'profile' => [],
            'app_name' => 'Empty Profile Test',
            'hostname' => 'test',
            'date' => 1714289304,
        ];

        // Process through pipeline
        $preparePeaks = new PreparePeaks();
        $event = $preparePeaks->handle($originalEvent);

        $calculateDiffs = new CalculateDiffsBetweenEdges();
        $event = $calculateDiffs->handle($event);

        $prepareEdges = new PrepareEdges();
        $event = $prepareEdges->handle($event);

        // Verify empty profile handling
        $this->assertSame([], $event['profile']);
        $this->assertSame(0, $event['peaks']['cpu']);
        $this->assertSame(0, $event['peaks']['wt']);
        $this->assertSame([], $event['edges']);
        $this->assertSame(0, $event['total_edges']);

        // Original data should be preserved
        $this->assertSame('Empty Profile Test', $event['app_name']);
        $this->assertSame('test', $event['hostname']);
        $this->assertSame(1714289304, $event['date']);
    }

    public function testPipelineWithLargeCallStack(): void
    {
        // Create a deep call stack
        $profile = [];
        $callStack = ['main()', 'App\\Bootstrap', 'App\\Kernel', 'App\\Router', 'App\\Controller', 'App\\Service', 'App\\Repository', 'PDO::query', 'strlen'];
        // Build profile with nested calls
        $counter = count($callStack);

        // Build profile with nested calls
        for ($i = 0; $i < $counter; $i++) {
            $currentFunc = $callStack[$i];
            $metrics = [
                'cpu' => max(10, 1000 - ($i * 100)),
                'wt' => max(50, 5000 - ($i * 500)),
                'mu' => max(100, 50000 - ($i * 5000)),
                'pmu' => max(200, 60000 - ($i * 6000)),
                'ct' => $i === 8 ? 50 : 1, // strlen called many times
            ];

            if ($i === 0) {
                $profile[$currentFunc] = $metrics;
            } else {
                $parentFunc = $callStack[$i - 1];
                $profile["{$parentFunc}==>{$currentFunc}"] = $metrics;
            }
        }

        $originalEvent = [
            'profile' => $profile,
            'app_name' => 'Large Call Stack Test',
            'hostname' => 'performance-test',
            'date' => 1714289305,
        ];

        // Process through pipeline
        $preparePeaks = new PreparePeaks();
        $event = $preparePeaks->handle($originalEvent);

        $calculateDiffs = new CalculateDiffsBetweenEdges();
        $event = $calculateDiffs->handle($event);

        $prepareEdges = new PrepareEdges();
        $event = $prepareEdges->handle($event);

        // Verify large call stack handling
        $this->assertSame(9, $event['total_edges']);
        $this->assertCount(9, $event['edges']);

        // Verify peaks from main()
        $this->assertSame(1000, $event['peaks']['cpu']);
        $this->assertSame(5000, $event['peaks']['wt']);

        // Verify parent-child relationships are maintained
        $edges = $event['edges'];

        // Check that the deepest call (strlen) has the right parent chain
        $strlenEdge = $edges['e1']; // First after reverse
        $this->assertSame('strlen', $strlenEdge['callee']);
        $this->assertSame('PDO::query', $strlenEdge['caller']);
        $this->assertSame('e2', $strlenEdge['parent']);

        $pdoEdge = $edges['e2'];
        $this->assertSame('PDO::query', $pdoEdge['callee']);
        $this->assertSame('App\\Repository', $pdoEdge['caller']);
        $this->assertSame('e3', $pdoEdge['parent']);

        // Check that main() has no parent
        $mainEdge = $edges['e9']; // Last after processing
        $this->assertSame('main()', $mainEdge['callee']);
        $this->assertNull($mainEdge['caller']);
        $this->assertNull($mainEdge['parent']);

        // Verify percentage calculations work correctly for deep stack
        $this->assertSame(10.0, $strlenEdge['cost']['p_cpu']); // 100/1000 * 100
        $this->assertSame(100.0, $mainEdge['cost']['p_cpu']); // 1000/1000 * 100
    }

    public function testPipelinePreservesAllOriginalData(): void
    {
        $originalEvent = [
            'profile' => [
                'main()' => ['cpu' => 1000, 'wt' => 2000, 'mu' => 10000, 'pmu' => 15000, 'ct' => 1],
            ],
            'app_name' => 'Data Preservation Test',
            'hostname' => 'data-test-host',
            'tags' => ['env' => 'test', 'version' => '2.0', 'user' => 'tester'],
            'date' => 1714289306,
            'custom_field' => 'custom_value',
            'nested' => [
                'deep' => [
                    'value' => 'preserved',
                ],
            ],
        ];

        // Process through complete pipeline
        $handlers = [
            new PreparePeaks(),
            new CalculateDiffsBetweenEdges(),
            new PrepareEdges(),
        ];

        $event = $originalEvent;
        foreach ($handlers as $handler) {
            $event = $handler->handle($event);
        }

        // Verify all original data is preserved
        $this->assertSame('Data Preservation Test', $event['app_name']);
        $this->assertSame('data-test-host', $event['hostname']);
        $this->assertSame(['env' => 'test', 'version' => '2.0', 'user' => 'tester'], $event['tags']);
        $this->assertSame(1714289306, $event['date']);
        $this->assertSame('custom_value', $event['custom_field']);
        $this->assertSame(['deep' => ['value' => 'preserved']], $event['nested']);

        // Verify new data was added
        $this->assertArrayHasKey('peaks', $event);
        $this->assertArrayHasKey('edges', $event);
        $this->assertArrayHasKey('total_edges', $event);

        // Verify original profile data was preserved alongside processing
        $this->assertArrayHasKey('profile', $event);
        $this->assertArrayHasKey('main()', $event['profile']);
        $this->assertSame(1000, $event['profile']['main()']['cpu']);
    }
}
