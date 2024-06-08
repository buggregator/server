<?php

declare(strict_types=1);

namespace Modules\Profiler\Interfaces\Queries;

use Cycle\ORM\ORMInterface;
use Modules\Profiler\Application\Query\FindTopFunctionsByUuid;
use Modules\Profiler\Domain\Edge;
use Modules\Profiler\Domain\Profile;
use Spiral\Cqrs\Attribute\QueryHandler;

// TODO: refactor this, use repository
final class FindTopFunctionsByUuidHandler
{
    public function __construct(
        private ORMInterface $orm,
    ) {}

    #[QueryHandler]
    public function __invoke(FindTopFunctionsByUuid $query): array
    {
        $profile = $this->orm->getRepository(Profile::class)->findByPK($query->profileUuid);

        $overallTotals = [];

        $functions = [];

        /** @var Edge[] $edges */
        $edges = $profile->edges;

        $metrics = ['cpu', 'ct', 'wt', 'mu', 'pmu'];

        foreach ($metrics as $metric) {
            $overallTotals[$metric] = 0;
        }

        foreach ($edges as $edge) {
            if (!isset($functions[$edge->getCallee()])) {
                $functions[$edge->getCallee()] = [
                    'function' => $edge->getCallee(),
                ];

                foreach ($metrics as $metric) {
                    $functions[$edge->getCallee()][$metric] = $edge->getCost()->{$metric};
                }
                continue;
            }

            foreach ($metrics as $metric) {
                $overallTotals[$metric] = $functions['main()'][$metric];
            }
        }

        foreach ($functions as $function => $m) {
            foreach ($metrics as $metric) {
                $functions[$function]['excl_' . $metric] = $functions[$function][$metric];
            }

            $overallTotals['ct'] += $m['ct'];
        }

        foreach ($edges as $edge) {
            if (!$edge->getCaller()) {
                continue;
            }

            foreach ($metrics as $metric) {
                $field = 'excl_' . $metric;
                $functions[$edge->getCaller()][$field] -= $edge->getCost()->{$metric};

                if ($functions[$edge->getCaller()][$field] < 0) {
                    $functions[$edge->getCaller()][$field] = 0;
                }
            }
        }

        $sortMetric = $query->metric->value;
        \usort($functions, static fn(array $a, array $b) => $b[$sortMetric] <=> $a[$sortMetric]);

        $functions = \array_slice($functions, 0, $query->limit);

        foreach ($functions as $function => $m) {
            foreach ($metrics as $metric) {
                $functions[$function]['p_' . $metric] = \round(
                    $functions[$function][$metric] / $overallTotals[$metric] * 100,
                    3,
                );
                $functions[$function]['p_excl_' . $metric] = \round(
                    $functions[$function]['excl_' . $metric] / $overallTotals[$metric] * 100,
                    3,
                );
            }
        }

        return [
            'overall_totals' => $overallTotals,
            'functions' => $functions,
        ];
    }
}
