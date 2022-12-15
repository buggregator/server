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

        $id = 1;
        foreach ($data as $name => $values) {
            [$parent, $func] = $this->splitName($name);
            $values = \array_merge($values, [
                'p_cpu' => round($values['cpu'] > 0 ? ($values['cpu'] / $event['peaks']['cpu'] * 100) : 0, 2),
                'p_mu' => round($values['mu'] > 0 ? ($values['mu'] / $event['peaks']['mu'] * 100) : 0, 2),
                'p_pmu' => round($values['pmu'] > 0 ? ($values['pmu'] / $event['peaks']['pmu'] * 100) : 0, 2),
                'p_wt' => round($values['wt'] > 0 ? ($values['wt'] / $event['peaks']['wt'] * 100) : 0, 2),
            ]);
            $event['edges']['e' . $id] = [
                'caller' => $parent,
                'callee' => $func,
                'cost' => $values,
            ];

            $id++;
        }

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
