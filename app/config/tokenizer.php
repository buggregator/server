<?php

declare(strict_types=1);

return [
    'directories' => [
        directory('app'),
        directory('modules'),
    ],
    'exclude' => [
        directory('resources'),
        directory('config'),
        'tests',
        'migrations',
    ],
];
