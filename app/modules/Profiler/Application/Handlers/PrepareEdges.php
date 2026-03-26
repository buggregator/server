<?php

declare(strict_types=1);

namespace Modules\Profiler\Application\Handlers;

use Modules\Profiler\Application\EventHandlerInterface;

final class PrepareEdges implements EventHandlerInterface
{
    public function handle(array $event): array
    {
        $profile = $event['profile'] ?? [];

        // First pass: create all edges and build callee → edgeId map
        $edgesTemp = [];
        $calleeToEdgeId = [];

        $id = 1;
        foreach ($profile as $name => $values) {
            [$caller, $callee] = EdgeNameSplitter::split($name);

            foreach (['cpu', 'mu', 'pmu', 'wt', 'ct'] as $key) {
                $peakValue = $event['peaks'][$key] ?? 0;
                $values['p_' . $key] = $peakValue > 0
                    ? \round(($values[$key] ?? 0) / $peakValue * 100, 3)
                    : 0;
            }

            $edgeId = 'e' . $id;
            $edgesTemp[$edgeId] = [
                'id' => $edgeId,
                'caller' => $caller,
                'callee' => $callee,
                'cost' => $values,
                'parent' => null,
            ];

            // Map callee name → first edge ID where it appears as callee
            if (!isset($calleeToEdgeId[$callee])) {
                $calleeToEdgeId[$callee] = $edgeId;
            }

            $id++;
        }

        // Second pass: resolve parent references using the complete callee map
        foreach ($edgesTemp as &$edge) {
            if ($edge['caller'] !== null && isset($calleeToEdgeId[$edge['caller']])) {
                $edge['parent'] = $calleeToEdgeId[$edge['caller']];
            }
        }
        unset($edge);

        // BFS ordering: parents always come before their children
        $childrenMap = [];
        $roots = [];
        foreach ($edgesTemp as $edgeId => $edge) {
            if ($edge['parent'] === null) {
                $roots[] = $edgeId;
            } else {
                $childrenMap[$edge['parent']][] = $edgeId;
            }
        }

        $event['edges'] = [];
        $queue = $roots;
        while ($queue) {
            $current = \array_shift($queue);
            $event['edges'][$current] = $edgesTemp[$current];
            foreach ($childrenMap[$current] ?? [] as $childId) {
                $queue[] = $childId;
            }
        }

        // Add any orphaned edges (data inconsistency fallback)
        foreach ($edgesTemp as $edgeId => $edge) {
            if (!isset($event['edges'][$edgeId])) {
                $event['edges'][$edgeId] = $edge;
            }
        }

        $event['total_edges'] = \count($event['edges']);

        return $event;
    }

}
