<?php

declare(strict_types=1);

namespace Modules\Profiler\Interfaces\Queries;

use Cycle\ORM\ORMInterface;
use Modules\Profiler\Application\Query\FindProfileSummaryByUuid;
use Modules\Profiler\Domain\Edge;
use Modules\Profiler\Domain\Profile;
use Spiral\Cqrs\Attribute\QueryHandler;

final class FindProfileSummaryByUuidHandler
{
    public function __construct(
        private ORMInterface $orm,
    ) {}

    #[QueryHandler]
    public function __invoke(FindProfileSummaryByUuid $query): array
    {
        $profile = $this->orm->getRepository(Profile::class)->findByPK($query->profileUuid);

        $functions = [];
        $metrics = ['cpu', 'ct', 'wt', 'mu', 'pmu'];

        /** @var Edge[] $edges */
        $edges = $profile->edges;

        // Aggregate inclusive metrics per function
        foreach ($edges as $edge) {
            $callee = $edge->getCallee();

            if (!isset($functions[$callee])) {
                $functions[$callee] = ['function' => $callee];
                foreach ($metrics as $metric) {
                    $functions[$callee][$metric] = 0;
                }
            }

            foreach ($metrics as $metric) {
                $functions[$callee][$metric] += $edge->getCost()->{$metric};
            }
        }

        // Overall totals from main()
        $overallTotals = [];
        foreach ($metrics as $metric) {
            $overallTotals[$metric] = $functions['main()'][$metric] ?? 0;
        }
        $overallTotals['ct'] = 0;
        foreach ($functions as $m) {
            $overallTotals['ct'] += $m['ct'];
        }

        // Initialize exclusive metrics
        foreach (\array_keys($functions) as $function) {
            foreach ($metrics as $metric) {
                $functions[$function]['excl_' . $metric] = $functions[$function][$metric];
            }
        }

        // Subtract children's inclusive from parent's exclusive
        foreach ($edges as $edge) {
            if (!$edge->getCaller()) {
                continue;
            }

            foreach ($metrics as $metric) {
                $field = 'excl_' . $metric;
                if (!isset($functions[$edge->getCaller()][$field])) {
                    continue;
                }
                $functions[$edge->getCaller()][$field] -= $edge->getCost()->{$metric};
                if ($functions[$edge->getCaller()][$field] < 0) {
                    $functions[$edge->getCaller()][$field] = 0;
                }
            }
        }

        $fnList = \array_values($functions);

        // Slowest function by exclusive wall time
        \usort($fnList, static fn(array $a, array $b) => ($b['excl_wt'] ?? 0) <=> ($a['excl_wt'] ?? 0));
        $slowest = $fnList[0] ?? null;

        // Memory hotspot by exclusive memory usage
        \usort($fnList, static fn(array $a, array $b) => ($b['excl_mu'] ?? 0) <=> ($a['excl_mu'] ?? 0));
        $memoryHotspot = $fnList[0] ?? null;

        // Most called function (exclude main)
        $fnListNoMain = \array_filter($fnList, static fn(array $f) => $f['function'] !== 'main()');
        \usort($fnListNoMain, static fn(array $a, array $b) => ($b['ct'] ?? 0) <=> ($a['ct'] ?? 0));
        $mostCalled = \array_values($fnListNoMain)[0] ?? null;

        return [
            'overall_totals' => $overallTotals,
            'slowest_function' => $slowest ? [
                'function' => $slowest['function'],
                'excl_wt' => $slowest['excl_wt'],
                'p_excl_wt' => $overallTotals['wt'] > 0
                    ? \round($slowest['excl_wt'] / $overallTotals['wt'] * 100, 1)
                    : 0,
            ] : null,
            'memory_hotspot' => $memoryHotspot ? [
                'function' => $memoryHotspot['function'],
                'excl_mu' => $memoryHotspot['excl_mu'],
                'p_excl_mu' => $overallTotals['mu'] > 0
                    ? \round($memoryHotspot['excl_mu'] / $overallTotals['mu'] * 100, 1)
                    : 0,
            ] : null,
            'most_called' => $mostCalled ? [
                'function' => $mostCalled['function'],
                'ct' => $mostCalled['ct'],
            ] : null,
        ];
    }
}
