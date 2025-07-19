<?php

declare(strict_types=1);

namespace Tests\Unit\Modules\Profiler\Application\Handlers;

use Modules\Profiler\Application\Handlers\CalculateDiffsBetweenEdges;
use PHPUnit\Framework\TestCase;

final class CalculateDiffsBetweenEdgesTest extends TestCase
{
    private CalculateDiffsBetweenEdges $handler;

    protected function setUp(): void
    {
        $this->handler = new CalculateDiffsBetweenEdges();
    }

    public function testHandleWithCompleteMetrics(): void
    {
        $event = [
            'profile' => [
                'main()' => ['cpu' => 1000, 'wt' => 2000, 'mu' => 10000, 'pmu' => 15000, 'ct' => 1],
                'main()==>funcA' => ['cpu' => 300, 'wt' => 600, 'mu' => 3000, 'pmu' => 4000, 'ct' => 1],
                'funcA==>funcB' => ['cpu' => 100, 'wt' => 200, 'mu' => 1000, 'pmu' => 1500, 'ct' => 1],
            ],
        ];

        $result = $this->handler->handle($event);

        // First function (main) should not have diff values
        $this->assertArrayNotHasKey('d_cpu', $result['profile']['main()']);

        // Second function should have diff calculated against main()
        $mainToFuncA = $result['profile']['main()==>funcA'];
        $this->assertSame(700, $mainToFuncA['d_cpu']); // 1000 - 300
        $this->assertSame(1400, $mainToFuncA['d_wt']); // 2000 - 600
        $this->assertSame(7000, $mainToFuncA['d_mu']); // 10000 - 3000
        $this->assertSame(11000, $mainToFuncA['d_pmu']); // 15000 - 4000

        // Third function should have diff calculated against funcA
        $funcAToFuncB = $result['profile']['funcA==>funcB'];
        $this->assertSame(200, $funcAToFuncB['d_cpu']); // 300 - 100
        $this->assertSame(400, $funcAToFuncB['d_wt']); // 600 - 200
    }

    public function testHandleWithMissingCpuMetrics(): void
    {
        $event = [
            'profile' => [
                'main()' => ['wt' => 2000, 'mu' => 10000, 'pmu' => 15000, 'ct' => 1], // No CPU
                'main()==>funcA' => ['wt' => 600, 'mu' => 3000, 'pmu' => 4000, 'ct' => 1], // No CPU
            ],
        ];

        $result = $this->handler->handle($event);

        // Should not throw error and should calculate with CPU = 0
        $mainToFuncA = $result['profile']['main()==>funcA'];
        $this->assertSame(0, $mainToFuncA['d_cpu']); // 0 - 0 = 0
        $this->assertSame(1400, $mainToFuncA['d_wt']); // 2000 - 600
        $this->assertSame(7000, $mainToFuncA['d_mu']); // 10000 - 3000
        $this->assertSame(11000, $mainToFuncA['d_pmu']); // 15000 - 4000
    }

    public function testHandleWithMissingOtherMetrics(): void
    {
        $event = [
            'profile' => [
                'main()' => ['cpu' => 1000, 'wt' => 2000, 'ct' => 1], // Missing mu, pmu
                'main()==>funcA' => ['cpu' => 300, 'wt' => 600, 'ct' => 1], // Missing mu, pmu
            ],
        ];

        $result = $this->handler->handle($event);

        $mainToFuncA = $result['profile']['main()==>funcA'];
        $this->assertSame(700, $mainToFuncA['d_cpu']); // 1000 - 300
        $this->assertSame(1400, $mainToFuncA['d_wt']); // 2000 - 600
        $this->assertSame(0, $mainToFuncA['d_mu']); // 0 - 0 = 0
        $this->assertSame(0, $mainToFuncA['d_pmu']); // 0 - 0 = 0
    }

    public function testHandleWithEmptyProfile(): void
    {
        $event = ['profile' => []];

        $result = $this->handler->handle($event);

        $this->assertSame([], $result['profile']);
    }

    public function testHandleWithSingleFunction(): void
    {
        $event = [
            'profile' => [
                'main()' => ['cpu' => 1000, 'wt' => 2000, 'mu' => 10000, 'pmu' => 15000, 'ct' => 1],
            ],
        ];

        $result = $this->handler->handle($event);

        // Single function should not have diff values
        $this->assertArrayNotHasKey('d_cpu', $result['profile']['main()']);
        $this->assertArrayNotHasKey('d_wt', $result['profile']['main()']);
        $this->assertArrayNotHasKey('d_mu', $result['profile']['main()']);
        $this->assertArrayNotHasKey('d_pmu', $result['profile']['main()']);
    }

    public function testHandleWithComplexCallStack(): void
    {
        $event = [
            'profile' => [
                'main()' => ['cpu' => 1000, 'wt' => 2000, 'mu' => 10000, 'pmu' => 15000, 'ct' => 1],
                'main()==>classA::methodB' => ['cpu' => 300, 'wt' => 600, 'mu' => 3000, 'pmu' => 4000, 'ct' => 2],
                'classA::methodB==>classC::methodD' => [
                    'cpu' => 100,
                    'wt' => 200,
                    'mu' => 1000,
                    'pmu' => 1500,
                    'ct' => 1,
                ],
                'classC::methodD==>strlen' => ['cpu' => 10, 'wt' => 20, 'mu' => 100, 'pmu' => 150, 'ct' => 5],
            ],
        ];

        $result = $this->handler->handle($event);

        // Test the full chain of diff calculations
        $mainToClassA = $result['profile']['main()==>classA::methodB'];
        $this->assertSame(700, $mainToClassA['d_cpu']); // 1000 - 300

        $classAToClassC = $result['profile']['classA::methodB==>classC::methodD'];
        $this->assertSame(200, $classAToClassC['d_cpu']); // 300 - 100

        $classCToStrlen = $result['profile']['classC::methodD==>strlen'];
        $this->assertSame(90, $classCToStrlen['d_cpu']); // 100 - 10
    }

    public function testHandlePreservesOriginalValues(): void
    {
        $event = [
            'profile' => [
                'main()' => ['cpu' => 1000, 'wt' => 2000, 'mu' => 10000, 'pmu' => 15000, 'ct' => 1],
                'main()==>funcA' => ['cpu' => 300, 'wt' => 600, 'mu' => 3000, 'pmu' => 4000, 'ct' => 1],
            ],
        ];

        $result = $this->handler->handle($event);

        // Original values should be preserved alongside diff values
        $mainToFuncA = $result['profile']['main()==>funcA'];
        $this->assertSame(300, $mainToFuncA['cpu']); // Original value preserved
        $this->assertSame(600, $mainToFuncA['wt']); // Original value preserved
        $this->assertSame(700, $mainToFuncA['d_cpu']); // Diff value added
        $this->assertSame(1400, $mainToFuncA['d_wt']); // Diff value added
    }

    public function testHandleWithMixedCompleteAndIncompleteMetrics(): void
    {
        $event = [
            'profile' => [
                'main()' => ['cpu' => 1000, 'wt' => 2000, 'mu' => 10000, 'pmu' => 15000, 'ct' => 1],
                'main()==>funcA' => ['wt' => 600, 'mu' => 3000, 'ct' => 1], // Missing cpu, pmu
                'funcA==>funcB' => ['cpu' => 100, 'wt' => 200, 'ct' => 1], // Missing mu, pmu
            ],
        ];

        $result = $this->handler->handle($event);

        $mainToFuncA = $result['profile']['main()==>funcA'];
        $this->assertSame(1000, $mainToFuncA['d_cpu']); // 1000 - 0
        $this->assertSame(1400, $mainToFuncA['d_wt']); // 2000 - 600
        $this->assertSame(7000, $mainToFuncA['d_mu']); // 10000 - 3000
        $this->assertSame(15000, $mainToFuncA['d_pmu']); // 15000 - 0

        $funcAToFuncB = $result['profile']['funcA==>funcB'];
        $this->assertSame(-100, $funcAToFuncB['d_cpu']); // 0 - 100
        $this->assertSame(400, $funcAToFuncB['d_wt']); // 600 - 200
        $this->assertSame(3000, $funcAToFuncB['d_mu']); // 3000 - 0
        $this->assertSame(0, $funcAToFuncB['d_pmu']); // 0 - 0
    }
}
