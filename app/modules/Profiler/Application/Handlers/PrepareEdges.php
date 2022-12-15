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
