<?php

declare(strict_types=1);

use Spiral\Cache\Storage\ArrayStorage;
use Spiral\Cache\Storage\FileStorage;

return [
    'default' => env('CACHE_STORAGE', 'local'),
    'storages' => [
        'events' => [
            'type' => 'roadrunner',
            'driver' => 'events',
        ],
    ],
];
