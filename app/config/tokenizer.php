<?php

declare(strict_types=1);

return [
    'directories' => [
        directory('app') . '/src',
        directory('modules'),
    ],
    'exclude' => [
        directory('resources'),
        directory('config'),
        directory('vendor'),
        'tests',
    ],
];
