<?php

declare(strict_types=1);

namespace Modules\Profiler\Interfaces\Queries;

use Cycle\ORM\ORMInterface;
use Modules\Profiler\Application\Query\CompareProfiles;
use Modules\Profiler\Domain\Edge;
use Modules\Profiler\Domain\Profile;
use Spiral\Cqrs\Attribute\QueryHandler;

final class CompareProfilesHandler
{
    public function __construct(
        private ORMInterface $orm,
    ) {}

    #[QueryHandler]
    public function __invoke(CompareProfiles $query): array
    {
        $baseFunctions = $this->aggregateFunctions($query->baseProfileUuid);
        $compareFunctions = $this->aggregateFunctions($query->compareProfileUuid);

        $allFunctionNames = \array_unique(\array_merge(
            \array_keys($baseFunctions),
            \array_keys($compareFunctions),
        ));

        $metrics = ['cpu', 'wt', 'ct', 'mu', 'pmu', 'excl_cpu', 'excl_wt', 'excl_ct', 'excl_mu', 'excl_pmu'];
        $diff = [];

        foreach ($allFunctionNames as $fn) {
            $base = $baseFunctions[$fn] ?? null;
            $compare = $compareFunctions[$fn] ?? null;

            $row = ['function' => $fn];

            foreach ($metrics as $metric) {
                $baseVal = $base[$metric] ?? 0;
                $compareVal = $compare[$metric] ?? 0;
                $row['base_' . $metric] = $baseVal;
                $row['compare_' . $metric] = $compareVal;
                $row['diff_' . $metric] = $compareVal - $baseVal;
            }

            $diff[] = $row;
        }

        // Sort by absolute diff in exclusive wall time descending
        \usort($diff, static fn(array $a, array $b) => \abs($b['diff_excl_wt']) <=> \abs($a['diff_excl_wt']));

        return [
            'functions' => \array_slice($diff, 0, $query->limit),
        ];
    }

    private function aggregateFunctions(mixed $profileUuid): array
    {
        $profile = $this->orm->getRepository(Profile::class)->findByPK($profileUuid);
        $functions = [];
        $metrics = ['cpu', 'ct', 'wt', 'mu', 'pmu'];

        /** @var Edge[] $edges */
        $edges = $profile->edges;

        foreach ($edges as $edge) {
            $callee = $edge->getCallee();
            if (!isset($functions[$callee])) {
                $functions[$callee] = [];
                foreach ($metrics as $metric) {
                    $functions[$callee][$metric] = 0;
                }
            }
            foreach ($metrics as $metric) {
                $functions[$callee][$metric] += $edge->getCost()->{$metric};
            }
        }

        // Exclusive metrics
        foreach (\array_keys($functions) as $fn) {
            foreach ($metrics as $metric) {
                $functions[$fn]['excl_' . $metric] = $functions[$fn][$metric];
            }
        }

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

        return $functions;
    }
}
