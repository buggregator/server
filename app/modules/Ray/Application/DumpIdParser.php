<?php

declare(strict_types=1);

namespace Modules\Ray\Application;

// todo: give a better name
final class DumpIdParser
{
    public static function find(string $html): ?string
    {
        // Regex to find all instances of sf-dump- followed by digits
        $pattern = '/sf-dump-\d+/';
        // Perform the search
        \preg_match($pattern, $html, $matches);

        return $matches[0] ?? null;
    }
}
