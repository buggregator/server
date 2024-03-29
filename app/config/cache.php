<?php

declare(strict_types=1);

use Spiral\Cache\Storage\ArrayStorage;

$defaultStorage = env('CACHE_DEFAULT_STORAGE', 'roadrunner');

return [
    'default' => env('CACHE_STORAGE', 'roadrunner'),
    'aliases' => [
        'events' => ['storage' => $defaultStorage, 'prefix' => 'events:'],
        'local' => ['storage' => $defaultStorage, 'prefix' => 'local:'],
    ],
    'storages' => [
        'array' => [
            'type' => ArrayStorage::class,
        ],
        'roadrunner' => [
            'type' => 'roadrunner',
            'driver' => 'local',
        ],
    ],
];
