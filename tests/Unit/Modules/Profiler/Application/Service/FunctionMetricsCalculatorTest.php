<?php

declare(strict_types=1);

namespace Tests\Unit\Modules\Profiler\Application\Service;

use App\Application\Domain\ValueObjects\Uuid;
use Modules\Profiler\Application\Service\FunctionMetricsCalculator;
use Modules\Profiler\Domain\Edge;
use Modules\Profiler\Domain\Edge\Cost;
use Modules\Profiler\Domain\Edge\Diff;
use Modules\Profiler\Domain\Edge\Percents;
use Modules\Profiler\Domain\FunctionMetrics;
use PHPUnit\Framework\TestCase;

final class FunctionMetricsCalculatorTest extends TestCase
{
    private FunctionMetricsCalculator $calculator;

    protected function setUp(): void
    {
        $this->calculator = new FunctionMetricsCalculator();
    }

    public function testCalculateMetricsWithSingleFunction(): void
    {
        $edges = [
            $this->createEdge('main()', 1000, 2000, 1, null),
        ];

        [$functions, $overallTotals] = $this->calculator->calculateMetrics($edges);

        $this->assertCount(1, $functions);
        $this->assertArrayHasKey('main()', $functions);

        $mainFunction = $functions['main()'];
        $this->assertSame('main()', $mainFunction->function);
        $this->assertSame(1000, $mainFunction->inclusive->cpu);
        $this->assertSame(2000, $mainFunction->inclusive->wt);

        // Overall totals should match main() function
        $this->assertSame(1000, $overallTotals->cpu);
        $this->assertSame(2000, $overallTotals->wt);
    }

    public function testCalculateMetricsWithMultipleFunctions(): void
    {
        $edges = [
            $this->createEdge('main()', 1000, 2000, 1, null),
            $this->createEdge('funcA', 300, 600, 1, 'main'),
            $this->createEdge('funcB', 200, 400, 1, 'main'),
        ];

        [$functions, $overallTotals] = $this->calculator->calculateMetrics($edges);

        $this->assertCount(3, $functions);
        $this->assertArrayHasKey('main()', $functions);
        $this->assertArrayHasKey('funcA', $functions);
        $this->assertArrayHasKey('funcB', $functions);

        // Overall totals should come from main()
        $this->assertSame(1000, $overallTotals->cpu);
        $this->assertSame(2000, $overallTotals->wt);

        // Check exclusive metrics (main should have child costs subtracted)
        $mainFunction = $functions['main()'];
        $this->assertSame(500, $mainFunction->exclusive->cpu); // 1000 - 300 - 200
        $this->assertSame(1000, $mainFunction->exclusive->wt); // 2000 - 600 - 400
    }

    public function testCalculateMetricsWithoutMainFunction(): void
    {
        $edges = [
            $this->createEdge('funcA', 300, 600, 2, null),
            $this->createEdge('funcB', 200, 400, 3, null),
        ];

        [$functions, $overallTotals] = $this->calculator->calculateMetrics($edges);

        $this->assertCount(2, $functions);

        // Should calculate totals from max values and sum of calls
        $this->assertSame(300, $overallTotals->cpu); // max(300, 200)
        $this->assertSame(600, $overallTotals->wt);  // max(600, 400)
        $this->assertSame(5, $overallTotals->ct);    // 2 + 3
    }

    public function testCalculateMetricsWithNestedCalls(): void
    {
        $edges = [
            $this->createEdge('main()', 1000, 2000, 1, null),
            $this->createEdge('parentFunc', 600, 1200, 1, 'main'),
            $this->createEdge('childFunc', 200, 400, 1, 'parentFunc'),
        ];

        [$functions, $overallTotals] = $this->calculator->calculateMetrics($edges);

        $this->assertCount(3, $functions);

        // Check exclusive calculations
        $mainFunction = $functions['main()'];
        $parentFunction = $functions['parentFunc'];
        $childFunction = $functions['childFunc'];

        // main() exclusive: 1000 - 600 = 400
        $this->assertSame(400, $mainFunction->exclusive->cpu);

        // parentFunc exclusive: 600 - 200 = 400
        $this->assertSame(400, $parentFunction->exclusive->cpu);

        // childFunc has no children, so exclusive = inclusive
        $this->assertSame(200, $childFunction->exclusive->cpu);
    }

    public function testSortFunctions(): void
    {
        $functions = [
            new FunctionMetrics('funcA', new Cost(100, 200, 1, 1000, 2000), new Cost(100, 200, 1, 1000, 2000)),
            new FunctionMetrics('funcB', new Cost(300, 600, 2, 3000, 6000), new Cost(300, 600, 2, 3000, 6000)),
            new FunctionMetrics('funcC', new Cost(200, 400, 1, 2000, 4000), new Cost(200, 400, 1, 2000, 4000)),
        ];

        $sorted = $this->calculator->sortFunctions($functions, 'cpu');

        $this->assertSame('funcB', $sorted[0]->function); // 300 cpu
        $this->assertSame('funcC', $sorted[1]->function); // 200 cpu
        $this->assertSame('funcA', $sorted[2]->function); // 100 cpu
    }

    public function testSortFunctionsByExclusiveMetric(): void
    {
        $functions = [
            new FunctionMetrics('funcA', new Cost(100, 200, 1, 1000, 2000), new Cost(50, 100, 1, 500, 1000)),
            new FunctionMetrics('funcB', new Cost(300, 600, 2, 3000, 6000), new Cost(250, 500, 2, 2500, 5000)),
            new FunctionMetrics('funcC', new Cost(200, 400, 1, 2000, 4000), new Cost(200, 400, 1, 2000, 4000)),
        ];

        $sorted = $this->calculator->sortFunctions($functions, 'excl_cpu');

        $this->assertSame('funcB', $sorted[0]->function); // 250 exclusive cpu
        $this->assertSame('funcC', $sorted[1]->function); // 200 exclusive cpu
        $this->assertSame('funcA', $sorted[2]->function); // 50 exclusive cpu
    }

    public function testToArrayFormat(): void
    {
        $functions = [
            new FunctionMetrics(
                'testFunc',
                new Cost(100, 200, 2, 1000, 2000),
                new Cost(80, 160, 1, 800, 1600),
            ),
        ];

        $overallTotals = new Cost(1000, 2000, 10, 10000, 20000);

        $result = $this->calculator->toArrayFormat($functions, $overallTotals);

        $this->assertCount(1, $result);
        $funcData = $result[0];

        $this->assertSame('testFunc', $funcData['function']);
        $this->assertSame(100, $funcData['cpu']);
        $this->assertSame(80, $funcData['excl_cpu']);
        $this->assertSame(10.0, $funcData['p_cpu']); // 100/1000 * 100
        $this->assertSame(8.0, $funcData['p_excl_cpu']); // 80/1000 * 100
    }

    public function testCalculateMetricsWithMissingCpuMetrics(): void
    {
        $edges = [
            $this->createEdgeWithoutCpu('main()', 2000, 1, null),
            $this->createEdgeWithoutCpu('funcA', 600, 1, 'main'),
        ];

        [$functions, $overallTotals] = $this->calculator->calculateMetrics($edges);

        $this->assertCount(2, $functions);

        // CPU should be 0 for all functions
        $mainFunction = $functions['main()'];
        $this->assertSame(0, $mainFunction->inclusive->cpu);
        $this->assertSame(0, $mainFunction->exclusive->cpu);
        $this->assertSame(0, $overallTotals->cpu);

        // Other metrics should work normally
        $this->assertSame(2000, $mainFunction->inclusive->wt);
        $this->assertSame(1400, $mainFunction->exclusive->wt); // 2000 - 600
    }

    private function createEdge(
        string $callee,
        int $cpu,
        int $wt,
        int $ct,
        ?string $caller,
    ): Edge {
        return new Edge(
            uuid: Uuid::generate(),
            profileUuid: Uuid::generate(),
            order: 1,
            cost: new Cost(cpu: $cpu, wt: $wt, ct: $ct, mu: $cpu * 10, pmu: $cpu * 20),
            diff: new Diff(cpu: 0, wt: 0, ct: 0, mu: 0, pmu: 0),
            percents: new Percents(cpu: 0.0, wt: 0.0, ct: 0.0, mu: 0.0, pmu: 0.0),
            callee: $callee,
            caller: $caller,
        );
    }

    private function createEdgeWithoutCpu(
        string $callee,
        int $wt,
        int $ct,
        ?string $caller,
    ): Edge {
        return new Edge(
            uuid: Uuid::generate(),
            profileUuid: Uuid::generate(),
            order: 1,
            cost: new Cost(cpu: 0, wt: $wt, ct: $ct, mu: $wt / 2, pmu: $wt),
            diff: new Diff(cpu: 0, wt: 0, ct: 0, mu: 0, pmu: 0),
            percents: new Percents(cpu: 0.0, wt: 0.0, ct: 0.0, mu: 0.0, pmu: 0.0),
            callee: $callee,
            caller: $caller,
        );
    }
}
