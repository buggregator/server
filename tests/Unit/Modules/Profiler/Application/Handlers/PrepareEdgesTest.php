<?php

declare(strict_types=1);

namespace Tests\Unit\Modules\Profiler\Application\Handlers;

use Modules\Profiler\Application\Handlers\PrepareEdges;
use PHPUnit\Framework\TestCase;

final class PrepareEdgesTest extends TestCase
{
    private PrepareEdges $handler;

    protected function setUp(): void
    {
        $this->handler = new PrepareEdges();
    }

    public function testHandleWithCompleteData(): void
    {
        $event = [
            'profile' => [
                'main()' => ['cpu' => 1000, 'wt' => 2000, 'mu' => 10000, 'pmu' => 15000, 'ct' => 1],
                'main()==>funcA' => ['cpu' => 300, 'wt' => 600, 'mu' => 3000, 'pmu' => 4000, 'ct' => 1],
                'funcA==>funcB' => ['cpu' => 100, 'wt' => 200, 'mu' => 1000, 'pmu' => 1500, 'ct' => 2],
            ],
            'peaks' => ['cpu' => 1000, 'wt' => 2000, 'mu' => 10000, 'pmu' => 15000, 'ct' => 1],
        ];

        $result = $this->handler->handle($event);

        dump($result);

        $this->assertArrayHasKey('edges', $result);
        $this->assertArrayHasKey('total_edges', $result);
        $this->assertSame(3, $result['total_edges']);

        $edges = $result['edges'];
        $this->assertCount(3, $edges);

        // Check first edge (main)
        $edge1 = $edges['e1'];
        $this->assertSame('e1', $edge1['id']);
        $this->assertSame('main()', $edge1['callee']);
        $this->assertNull($edge1['caller']);
        $this->assertNull($edge1['parent']);

        // Check percentage calculations
        $this->assertSame(100.0, $edge1['cost']['p_cpu']); // 1000/1000 * 100
        $this->assertSame(100.0, $edge1['cost']['p_wt']); // 2000/2000 * 100

        // Check second edge (main ==> funcA)
        $edge2 = $edges['e2'];
        $this->assertSame('funcA', $edge2['callee']);
        $this->assertSame('main()', $edge2['caller']);
        $this->assertSame('e1', $edge2['parent']);
        $this->assertSame(30.0, $edge2['cost']['p_cpu']); // 300/1000 * 100

        // Check third edge (funcA ==> funcB)
        $edge3 = $edges['e3'];
        $this->assertSame('funcB', $edge3['callee']);
        $this->assertSame('funcA', $edge3['caller']);
        $this->assertSame('e2', $edge3['parent']);
        $this->assertSame(10.0, $edge3['cost']['p_cpu']); // 100/1000 * 100
    }

    public function testHandleWithMissingCpuMetrics(): void
    {
        $event = [
            'profile' => [
                'main()' => ['wt' => 2000, 'mu' => 10000, 'pmu' => 15000, 'ct' => 1], // No CPU
                'main()==>funcA' => ['wt' => 600, 'mu' => 3000, 'pmu' => 4000, 'ct' => 1], // No CPU
            ],
            'peaks' => ['cpu' => 0, 'wt' => 2000, 'mu' => 10000, 'pmu' => 15000, 'ct' => 1],
        ];

        $result = $this->handler->handle($event);

        $edges = $result['edges'];
        $edge1 = $edges['e1'];
        $edge2 = $edges['e2'];

        // CPU should be handled gracefully with 0 values
        $this->assertSame(0.0, $edge1['cost']['p_cpu']);
        $this->assertSame(0.0, $edge2['cost']['p_cpu']);

        // Other metrics should work normally
        $this->assertSame(100.0, $edge1['cost']['p_wt']); // 2000/2000 * 100
        $this->assertSame(30.0, $edge2['cost']['p_wt']); // 600/2000 * 100
    }

    public function testHandleWithZeroPeaks(): void
    {
        $event = [
            'profile' => [
                'main()' => ['cpu' => 1000, 'wt' => 2000, 'mu' => 10000, 'pmu' => 15000, 'ct' => 1],
            ],
            'peaks' => ['cpu' => 0, 'wt' => 0, 'mu' => 0, 'pmu' => 0, 'ct' => 0], // All zero peaks
        ];

        $result = $this->handler->handle($event);

        $edges = $result['edges'];
        $edge1 = $edges['e1'];

        // Should handle division by zero gracefully
        $this->assertSame(0.0, $edge1['cost']['p_cpu']);
        $this->assertSame(0.0, $edge1['cost']['p_wt']);
        $this->assertSame(0.0, $edge1['cost']['p_mu']);
        $this->assertSame(0.0, $edge1['cost']['p_pmu']);
    }

    public function testHandleWithMissingPeaks(): void
    {
        $event = [
            'profile' => [
                'main()' => ['cpu' => 1000, 'wt' => 2000, 'mu' => 10000, 'pmu' => 15000, 'ct' => 1],
            ],
            // No peaks key
        ];

        $result = $this->handler->handle($event);

        $edges = $result['edges'];
        $edge1 = $edges['e1'];

        // Should handle missing peaks gracefully with defaults
        $this->assertSame(0.0, $edge1['cost']['p_cpu']);
        $this->assertSame(0.0, $edge1['cost']['p_wt']);
        $this->assertSame(0.0, $edge1['cost']['p_mu']);
        $this->assertSame(0.0, $edge1['cost']['p_pmu']);
    }

    public function testHandleWithSingleFunction(): void
    {
        $event = [
            'profile' => [
                'main()' => ['cpu' => 1000, 'wt' => 2000, 'mu' => 10000, 'pmu' => 15000, 'ct' => 1],
            ],
            'peaks' => ['cpu' => 1000, 'wt' => 2000, 'mu' => 10000, 'pmu' => 15000, 'ct' => 1],
        ];

        $result = $this->handler->handle($event);

        $this->assertSame(1, $result['total_edges']);

        $edges = $result['edges'];
        $this->assertCount(1, $edges);

        $edge1 = $edges['e1'];
        $this->assertSame('main()', $edge1['callee']);
        $this->assertNull($edge1['caller']);
        $this->assertNull($edge1['parent']);
        $this->assertSame(100.0, $edge1['cost']['p_cpu']);
    }

    public function testHandleWithEmptyProfile(): void
    {
        $event = [
            'profile' => [],
            'peaks' => ['cpu' => 0, 'wt' => 0, 'mu' => 0, 'pmu' => 0, 'ct' => 0],
        ];

        $result = $this->handler->handle($event);

        $this->assertSame(0, $result['total_edges']);
        $this->assertSame([], $result['edges']);
    }

    public function testHandlePreservesOtherEventData(): void
    {
        $event = [
            'profile' => [
                'main()' => ['cpu' => 1000, 'wt' => 2000, 'mu' => 10000, 'pmu' => 15000, 'ct' => 1],
            ],
            'peaks' => ['cpu' => 1000, 'wt' => 2000, 'mu' => 10000, 'pmu' => 15000, 'ct' => 1],
            'app_name' => 'Test App',
            'hostname' => 'localhost',
            'tags' => ['env' => 'test'],
        ];

        $result = $this->handler->handle($event);

        // Should preserve all original data
        $this->assertSame('Test App', $result['app_name']);
        $this->assertSame('localhost', $result['hostname']);
        $this->assertSame(['env' => 'test'], $result['tags']);
        $this->assertArrayHasKey('profile', $result);
        $this->assertArrayHasKey('peaks', $result);

        // And add new data
        $this->assertArrayHasKey('edges', $result);
        $this->assertArrayHasKey('total_edges', $result);
    }

    public function testHandleWithComplexFunctionNames(): void
    {
        $event = [
            'profile' => [
                'Namespace\\Class::method' => ['cpu' => 1000, 'wt' => 2000, 'mu' => 10000, 'pmu' => 15000, 'ct' => 1],
                'Namespace\\Class::method==>Another\\Class::staticMethod' => [
                    'cpu' => 300,
                    'wt' => 600,
                    'mu' => 3000,
                    'pmu' => 4000,
                    'ct' => 2,
                ],
                'Another\\Class::staticMethod==>strlen' => [
                    'cpu' => 50,
                    'wt' => 100,
                    'mu' => 500,
                    'pmu' => 600,
                    'ct' => 10,
                ],
            ],
            'peaks' => ['cpu' => 1000, 'wt' => 2000, 'mu' => 10000, 'pmu' => 15000, 'ct' => 1],
        ];

        $result = $this->handler->handle($event);

        $edges = $result['edges'];

        $this->assertSame('Namespace\\Class::method', $edges['e1']['callee']);
        $this->assertNull($edges['e1']['caller']);

        $this->assertSame('Another\\Class::staticMethod', $edges['e2']['callee']);
        $this->assertSame('Namespace\\Class::method', $edges['e2']['caller']);

        $this->assertSame('strlen', $edges['e3']['callee']);
        $this->assertSame('Another\\Class::staticMethod', $edges['e3']['caller']);
    }

    public function testHandleWithIncompleteMetricsInProfile(): void
    {
        $event = [
            'profile' => [
                'main()' => ['cpu' => 1000, 'wt' => 2000, 'ct' => 1], // Missing mu, pmu
                'main()==>funcA' => ['wt' => 600, 'mu' => 3000, 'ct' => 1], // Missing cpu, pmu
            ],
            'peaks' => ['cpu' => 1000, 'wt' => 2000, 'mu' => 10000, 'pmu' => 15000, 'ct' => 1],
        ];

        $result = $this->handler->handle($event);

        $edges = $result['edges'];

        // Should handle missing metrics by using defaults (0)
        $edge1 = $edges['e1'];
        $this->assertSame(100.0, $edge1['cost']['p_cpu']); // 1000/1000 * 100
        $this->assertSame(0.0, $edge1['cost']['p_mu']); // 0/10000 * 100
        $this->assertSame(0.0, $edge1['cost']['p_pmu']); // 0/15000 * 100

        $edge2 = $edges['e2'];
        $this->assertSame(0.0, $edge2['cost']['p_cpu']); // 0/1000 * 100
        $this->assertSame(30.0, $edge2['cost']['p_wt']); // 600/2000 * 100
        $this->assertSame(30.0, $edge2['cost']['p_mu']); // 3000/10000 * 100
        $this->assertSame(0.0, $edge2['cost']['p_pmu']); // 0/15000 * 100
    }

    public function testHandleReversingOrder(): void
    {
        // Test that array_reverse is working properly
        $event = [
            'profile' => [
                'first' => ['cpu' => 100, 'wt' => 200, 'mu' => 1000, 'pmu' => 1500, 'ct' => 1],
                'second' => ['cpu' => 200, 'wt' => 400, 'mu' => 2000, 'pmu' => 3000, 'ct' => 1],
                'third' => ['cpu' => 300, 'wt' => 600, 'mu' => 3000, 'pmu' => 4500, 'ct' => 1],
            ],
            'peaks' => ['cpu' => 1000, 'wt' => 2000, 'mu' => 10000, 'pmu' => 15000, 'ct' => 1],
        ];

        $result = $this->handler->handle($event);

        $edges = $result['edges'];

        // After reversing, 'third' should be processed first (e1), then 'second' (e2), then 'first' (e3)
        $this->assertSame('third', $edges['e1']['callee']);
        $this->assertSame('second', $edges['e2']['callee']);
        $this->assertSame('first', $edges['e3']['callee']);
    }
}
