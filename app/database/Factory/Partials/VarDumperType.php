<?php

declare(strict_types=1);

namespace Database\Factory\Partials;

trait VarDumperType
{
    protected static function getVarDumperPayload(): array
    {
        return [
            'payload' => [
                'type' => 'string',
                'value' => 'foo',
            ],
            'context' => [
                'timestamp' => 1701499845.012236,
                'source' => [
                    'name' => 'SomeClass.php',
                    'file' => '/app/src/SomeClass.php',
                    'line' => 16,
                    'file_excerpt' => false,
                ],
            ],
        ];
    }
}
