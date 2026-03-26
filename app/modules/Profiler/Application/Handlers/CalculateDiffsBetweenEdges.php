<?php

declare(strict_types=1);

namespace Modules\Profiler\Application\Handlers;

use Modules\Profiler\Application\EventHandlerInterface;

final class CalculateDiffsBetweenEdges implements EventHandlerInterface
{
    public function handle(array $event): array
    {
        $profile = $event['profile'] ?? [];

        // Aggregate children's inclusive metrics per parent function
        $childrenSum = [];
        foreach ($profile as $name => $values) {
            [$parent] = EdgeNameSplitter::split($name);

            if ($parent !== null) {
                if (!isset($childrenSum[$parent])) {
                    $childrenSum[$parent] = ['cpu' => 0, 'wt' => 0, 'mu' => 0, 'pmu' => 0, 'ct' => 0];
                }

                foreach (['cpu', 'wt', 'mu', 'pmu', 'ct'] as $metric) {
                    $childrenSum[$parent][$metric] += $values[$metric] ?? 0;
                }
            }
        }

        // Calculate diff: parent's inclusive minus sum of all its children (exclusive time of parent)
        foreach ($profile as $name => $values) {
            [, $func] = EdgeNameSplitter::split($name);
            $children = $childrenSum[$func] ?? ['cpu' => 0, 'wt' => 0, 'mu' => 0, 'pmu' => 0, 'ct' => 0];

            $event['profile'][$name] = \array_merge($values, [
                'd_cpu' => \max(0, ($values['cpu'] ?? 0) - $children['cpu']),
                'd_wt' => \max(0, ($values['wt'] ?? 0) - $children['wt']),
                'd_mu' => \max(0, ($values['mu'] ?? 0) - $children['mu']),
                'd_pmu' => \max(0, ($values['pmu'] ?? 0) - $children['pmu']),
                'd_ct' => \max(0, ($values['ct'] ?? 0) - $children['ct']),
            ]);
        }

        return $event;
    }
}
