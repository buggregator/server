<?php

declare(strict_types=1);

namespace Modules\Profiler\Application\Handlers;

final class EdgeNameSplitter
{
    /**
     * Split XHProf edge name "parent==>child" into [parent, child].
     * For root entries like "main()", returns [null, "main()"].
     *
     * @return array{0: string|null, 1: string}
     */
    public static function split(string $name): array
    {
        $parts = \explode('==>', $name);

        if (isset($parts[1])) {
            return [$parts[0], $parts[1]];
        }

        return [null, $parts[0]];
    }
}
