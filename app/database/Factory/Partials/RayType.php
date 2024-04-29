<?php

declare(strict_types=1);

namespace Database\Factory\Partials;

trait RayType
{
    protected static function getRayPayload(): array
    {
        return [
            'uuid' => '11325003-b9cf-4c06-83d0-8a18fe368ac4',
            'payloads' => [
                [
                    'type' => 'log',
                    'content' => [
                        'values' => ['foo'],
                        'meta' => [
                            ['clipboard_data' => 'foo',],
                        ],
                    ],
                    'origin' => [
                        'file' => '/app/src/SomeClass.php',
                        'line_number' => 13,
                        'hostname' => 'localhost',
                    ],
                ],
            ],
            'meta' => [
                'php_version' => '8.2.5',
                'php_version_id' => 80205,
                'project_name' => '',
                'ray_package_version' => '1.40.1.0',
            ],
        ];
    }
}
