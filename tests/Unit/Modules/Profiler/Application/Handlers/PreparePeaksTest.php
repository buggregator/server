<?php

declare(strict_types=1);

namespace Tests\Unit\Modules\Profiler\Application\Handlers;

use Modules\Profiler\Application\Handlers\PreparePeaks;
use PHPUnit\Framework\TestCase;

final class PreparePeaksTest extends TestCase
{
    private PreparePeaks $handler;

    protected function setUp(): void
    {
        $this->handler = new PreparePeaks();
    }

    public function testHandleWithCompleteMainFunction(): void
    {
        $event = [
            'profile' => [
                'main()' => ['cpu' => 1000, 'wt' => 2000, 'mu' => 10000, 'pmu' => 15000, 'ct' => 1],
                'main()==>funcA' => ['cpu' => 300, 'wt' => 600, 'mu' => 3000, 'pmu' => 4000, 'ct' => 1],
            ],
        ];

        $result = $this->handler->handle($event);

        $this->assertArrayHasKey('peaks', $result);
        $peaks = $result['peaks'];

        $this->assertSame(1000, $peaks['cpu']);
        $this->assertSame(2000, $peaks['wt']);
        $this->assertSame(1, $peaks['ct']);
        $this->assertSame(10000, $peaks['mu']);
        $this->assertSame(15000, $peaks['pmu']);
    }

    public function testHandleWithIncompleteMainFunction(): void
    {
        $event = [
            'profile' => [
                'main()' => ['wt' => 2000, 'mu' => 10000, 'ct' => 1], // Missing cpu and pmu
                'main()==>funcA' => ['cpu' => 300, 'wt' => 600, 'mu' => 3000, 'pmu' => 4000, 'ct' => 1],
            ],
        ];

        $result = $this->handler->handle($event);

        $peaks = $result['peaks'];

        $this->assertSame(0, $peaks['cpu']); // Default value
        $this->assertSame(2000, $peaks['wt']);
        $this->assertSame(1, $peaks['ct']);
        $this->assertSame(10000, $peaks['mu']);
        $this->assertSame(0, $peaks['pmu']); // Default value
    }

    public function testHandleWithoutMainFunction(): void
    {
        $event = [
            'profile' => [
                'funcA' => ['cpu' => 300, 'wt' => 600, 'mu' => 3000, 'pmu' => 4000, 'ct' => 1],
                'funcB' => ['cpu' => 200, 'wt' => 400, 'mu' => 2000, 'pmu' => 3000, 'ct' => 2],
            ],
        ];

        $result = $this->handler->handle($event);

        $peaks = $result['peaks'];

        // Should use default values when main() is not present
        $this->assertSame(0, $peaks['cpu']);
        $this->assertSame(0, $peaks['wt']);
        $this->assertSame(0, $peaks['ct']);
        $this->assertSame(0, $peaks['mu']);
        $this->assertSame(0, $peaks['pmu']);
    }

    public function testHandleWithEmptyProfile(): void
    {
        $event = ['profile' => []];

        $result = $this->handler->handle($event);

        $peaks = $result['peaks'];

        $this->assertSame(0, $peaks['cpu']);
        $this->assertSame(0, $peaks['wt']);
        $this->assertSame(0, $peaks['ct']);
        $this->assertSame(0, $peaks['mu']);
        $this->assertSame(0, $peaks['pmu']);
    }

    public function testHandleWithEmptyMainFunction(): void
    {
        $event = [
            'profile' => [
                'main()' => [], // Empty main function
                'funcA' => ['cpu' => 300, 'wt' => 600, 'mu' => 3000, 'pmu' => 4000, 'ct' => 1],
            ],
        ];

        $result = $this->handler->handle($event);

        $peaks = $result['peaks'];

        // Should use defaults for all metrics
        $this->assertSame(0, $peaks['cpu']);
        $this->assertSame(0, $peaks['wt']);
        $this->assertSame(0, $peaks['ct']);
        $this->assertSame(0, $peaks['mu']);
        $this->assertSame(0, $peaks['pmu']);
    }

    public function testHandlePreservesOtherEventData(): void
    {
        $event = [
            'profile' => [
                'main()' => ['cpu' => 1000, 'wt' => 2000, 'mu' => 10000, 'pmu' => 15000, 'ct' => 1],
            ],
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

        // And add peaks
        $this->assertArrayHasKey('peaks', $result);
        $this->assertSame(1000, $result['peaks']['cpu']);
    }

    public function testHandleWithMainFunctionContainingZeroValues(): void
    {
        $event = [
            'profile' => [
                'main()' => ['cpu' => 0, 'wt' => 0, 'mu' => 0, 'pmu' => 0, 'ct' => 0],
                'funcA' => ['cpu' => 300, 'wt' => 600, 'mu' => 3000, 'pmu' => 4000, 'ct' => 1],
            ],
        ];

        $result = $this->handler->handle($event);

        $peaks = $result['peaks'];

        // Should use the actual zero values from main(), not defaults
        $this->assertSame(0, $peaks['cpu']);
        $this->assertSame(0, $peaks['wt']);
        $this->assertSame(0, $peaks['ct']);
        $this->assertSame(0, $peaks['mu']);
        $this->assertSame(0, $peaks['pmu']);
    }

    public function testHandleWithMainFunctionHavingNegativeValues(): void
    {
        $event = [
            'profile' => [
                // Negative values shouldn't normally occur, but test robustness
                'main()' => ['cpu' => -100, 'wt' => 2000, 'mu' => -1000, 'pmu' => 15000, 'ct' => 1],
            ],
        ];

        $result = $this->handler->handle($event);

        $peaks = $result['peaks'];

        // Should preserve even negative values (let other parts of system handle validation)
        $this->assertSame(-100, $peaks['cpu']);
        $this->assertSame(2000, $peaks['wt']);
        $this->assertSame(-1000, $peaks['mu']);
        $this->assertSame(15000, $peaks['pmu']);
        $this->assertSame(1, $peaks['ct']);
    }

    public function testHandleWithMissingProfileKey(): void
    {
        $event = [
            'app_name' => 'Test App',
            'hostname' => 'localhost',
        ];

        $result = $this->handler->handle($event);

        $peaks = $result['peaks'];

        // Should handle missing profile key gracefully
        $this->assertSame(0, $peaks['cpu']);
        $this->assertSame(0, $peaks['wt']);
        $this->assertSame(0, $peaks['ct']);
        $this->assertSame(0, $peaks['mu']);
        $this->assertSame(0, $peaks['pmu']);
    }
}
