<?php

declare(strict_types=1);

namespace Modules\Profiler\Interfaces\Queries;

use Modules\Profiler\Domain\Edge;
use Cycle\ORM\ORMInterface;
use Modules\Profiler\Application\Query\FindTopFunctionsByUuid;
use Modules\Profiler\Application\Service\FunctionMetricsCalculator;
use Modules\Profiler\Domain\Profile;
use Spiral\Cqrs\Attribute\QueryHandler;

final readonly class FindTopFunctionsByUuidHandler
{
    public function __construct(
        private ORMInterface $orm,
        private FunctionMetricsCalculator $calculator,
    ) {}

    #[QueryHandler]
    public function __invoke(FindTopFunctionsByUuid $query): array
    {
        $profile = $this->orm->getRepository(Profile::class)->findByPK($query->profileUuid);

        /** @var Edge[] $edges */
        $edges = $profile->edges->toArray();

        // Calculate function metrics using domain service
        [$functions, $overallTotals] = $this->calculator->calculateMetrics($edges);

        // Sort functions by requested metric
        $sortedFunctions = $this->calculator->sortFunctions($functions, $query->metric->value);

        // Limit results
        $limitedFunctions = array_slice($sortedFunctions, 0, $query->limit);

        // Convert to API format
        $functionsArray = $this->calculator->toArrayFormat($limitedFunctions, $overallTotals);

        return [
            'schema' => $this->buildSchema(),
            'overall_totals' => $overallTotals->toArray(),
            'functions' => $functionsArray,
        ];
    }

    private function buildSchema(): array
    {
        return [
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
        ];
    }
}
