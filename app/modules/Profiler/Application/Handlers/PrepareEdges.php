<?php

declare(strict_types=1);

namespace Modules\Profiler\Application\Handlers;

use Modules\Profiler\Application\EventHandlerInterface;

final class PrepareEdges implements EventHandlerInterface
{
    public function handle(array $event): array
    {
        $data = \array_reverse($event['profile'] ?? []);

        $event['edges'] = [];
        $parents = [];

        $prev = null;

        $id = 1;
        foreach ($data as $name => $values) {
            [$parent, $func] = $this->splitName($name);

            $parentId = $parents[$parent] ?? $prev;

            foreach (['cpu', 'mu', 'pmu', 'wt'] as $key) {
                $values['p_' . $key] = \round(
                    $values[$key] > 0 ? ($values[$key]) / $event['peaks'][$key] * 100 : 0,
                    3,
                );
            }

            $event['edges']['e' . $id] = [
                'id' => 'e' . $id,
                'caller' => $parent,
                'callee' => $func,
                'cost' => $values,
                'parent' => $parentId,
            ];

            $parents[$func] = 'e' . $id;
            $prev = 'e' . $id;

            $id++;
        }

        $event['total_edges'] = \count($event['edges']);

        return $event;
    }

    /**
     * @return array{0: string|null, 1: string}
     */
    private function splitName(string $name): array
    {
        $a = \explode('==>', $name);
        if (isset($a[1])) {
            return $a;
        }

        return [null, $a[0]];
    }
}
