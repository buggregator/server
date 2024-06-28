<?php

declare(strict_types=1);

namespace Modules\Profiler\Interfaces\Queries;

use Cycle\ORM\ORMInterface;
use Modules\Profiler\Application\Query\FindTopFunctionsByUuid;
use Modules\Profiler\Domain\Edge;
use Modules\Profiler\Domain\Profile;
use Spiral\Cqrs\Attribute\QueryHandler;

// TODO: refactor this, use repository
final readonly class FindTopFunctionsByUuidHandler
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

                if (!isset($functions[$edge->getCaller()][$field])) {
                    $functions[$edge->getCaller()][$field] = 0;
                }

                $functions[$edge->getCaller()][$field] -= $edge->getCost()->{$metric};

                if ($functions[$edge->getCaller()][$field] < 0) {
                    $functions[$edge->getCaller()][$field] = 0;
                }
            }
        }

        $sortMetric = $query->metric->value;
        \usort($functions, static fn(array $a, array $b) => ($b[$sortMetric] ?? 0) <=> ($a[$sortMetric] ?? 0));

        $functions = \array_slice($functions, 0, $query->limit);

        foreach (array_keys($functions) as $function) {
            foreach ($metrics as $metric) {
                $functions[$function]['p_' . $metric] = \round(
                    $functions[$function][$metric] > 0 ? $functions[$function][$metric] / $overallTotals[$metric] * 100 : 0,
                    3,
                );
                $functions[$function]['p_excl_' . $metric] = \round(
                    $functions[$function]['excl_' . $metric] > 0 ? $functions[$function]['excl_' . $metric] / $overallTotals[$metric] * 100 : 0,
                    3,
                );
            }
        }

        return [
            'schema' => [
                [
                    'key' => 'function',
                    'label' => 'Function',
                    'description' => 'Function that was called',
                    'sortable' => false,
                    'values' => [
                        [
                            'key' => 'function',
                            'format' => 'string',
                        ],
                    ],
                ],
                [
                    'key' => 'ct',
                    'label' => 'CT',
                    'description' => 'Calls',
                    'sortable' => true,
                    'values' => [
                        [
                            'key' => 'ct',
                            'format' => 'number',
                        ],
                    ],
                ],
                [
                    'key' => 'cpu',
                    'label' => 'CPU',
                    'description' => 'CPU Time (ms)',
                    'sortable' => true,
                    'values' => [
                        [
                            'key' => 'cpu',
                            'format' => 'ms',
                        ],
                        [
                            'key' => 'p_cpu',
                            'format' => 'percent',
                            'type' => 'sub',
                        ],
                    ],
                ],
                [
                    'key' => 'excl_cpu',
                    'label' => 'CPU excl.',
                    'description' => 'CPU Time exclusions (ms)',
                    'sortable' => true,
                    'values' => [
                        [
                            'key' => 'excl_cpu',
                            'format' => 'ms',
                        ],
                        [
                            'key' => 'p_excl_cpu',
                            'format' => 'percent',
                            'type' => 'sub',
                        ],
                    ],
                ],
                [
                    'key' => 'wt',
                    'label' => 'WT',
                    'description' => 'Wall Time (ms)',
                    'sortable' => true,
                    'values' => [
                        [
                            'key' => 'wt',
                            'format' => 'ms',
                        ],
                        [
                            'key' => 'p_wt',
                            'format' => 'percent',
                            'type' => 'sub',
                        ],
                    ],
                ],
                [
                    'key' => 'excl_wt',
                    'label' => 'WT excl.',
                    'description' => 'Wall Time exclusions (ms)',
                    'sortable' => true,
                    'values' => [
                        [
                            'key' => 'excl_wt',
                            'format' => 'ms',
                        ],
                        [
                            'key' => 'p_excl_wt',
                            'format' => 'percent',
                            'type' => 'sub',
                        ],
                    ],
                ],
                [
                    'key' => 'mu',
                    'label' => 'MU',
                    'description' => 'Memory Usage (bytes)',
                    'sortable' => true,
                    'values' => [
                        [
                            'key' => 'mu',
                            'format' => 'bytes',
                        ],
                        [
                            'key' => 'p_mu',
                            'format' => 'percent',
                            'type' => 'sub',
                        ],
                    ],
                ],
                [
                    'key' => 'excl_mu',
                    'label' => 'MU excl.',
                    'description' => 'Memory Usage exclusions (bytes)',
                    'sortable' => true,
                    'values' => [
                        [
                            'key' => 'excl_mu',
                            'format' => 'bytes',
                        ],
                        [
                            'key' => 'p_excl_mu',
                            'format' => 'percent',
                            'type' => 'sub',
                        ],
                    ],
                ],
                [
                    'key' => 'pmu',
                    'label' => 'PMU',
                    'description' => 'Peak Memory Usage (bytes)',
                    'sortable' => true,
                    'values' => [
                        [
                            'key' => 'pmu',
                            'format' => 'bytes',
                        ],
                        [
                            'key' => 'p_pmu',
                            'format' => 'percent',
                            'type' => 'sub',
                        ],
                    ],
                ],
                [
                    'key' => 'excl_pmu',
                    'label' => 'PMU excl.',
                    'description' => 'Peak Memory Usage exclusions (bytes)',
                    'sortable' => true,
                    'values' => [
                        [
                            'key' => 'excl_pmu',
                            'format' => 'bytes',
                        ],
                        [
                            'key' => 'p_excl_pmu',
                            'format' => 'percent',
                            'type' => 'sub',
                        ],
                    ],
                ],
            ],
            'overall_totals' => $overallTotals,
            'functions' => $functions,
        ];
    }
}
