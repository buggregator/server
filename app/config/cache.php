<?php

declare(strict_types=1);

use Spiral\Cache\Storage\ArrayStorage;

$defaultStorage = env('CACHE_DEFAULT_STORAGE', 'roadrunner');

return [
    'default' => env('CACHE_STORAGE', 'roadrunner'),
    'aliases' => [
        'webhooks' => ['storage' => $defaultStorage, 'prefix' => 'webhooks:'],
        'local' => ['storage' => $defaultStorage, 'prefix' => 'local:'],
        'smtp' => ['storage' => $defaultStorage, 'prefix' => 'smtp:'],
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
