<?php

declare(strict_types=1);

use Modules\Webhooks\Interfaces\Job\WebhookHandler;
use Spiral\Queue\Driver\SyncDriver;
use Spiral\RoadRunner\Jobs\Queue\MemoryCreateInfo;
use Spiral\RoadRunnerBridge\Queue\Queue;

$defaultConnection = env('QUEUE_DEFAULT_CONNECTION', 'roadrunner');

return [
    'default' => env('QUEUE_CONNECTION', 'memory'),
    'aliases' => [
        'webhooks' => $defaultConnection,
        'events' => $defaultConnection,
    ],
    'pipelines' => [
        'memory' => [
            'connector' => new MemoryCreateInfo('local'),
            'consume' => true,
        ],
    ],
    'connections' => [
        'sync' => [
            'driver' => 'sync',
        ],
        'roadrunner' => [
            'driver' => 'roadrunner',
            'pipeline' => 'memory',
        ],
    ],
    'driverAliases' => [
        'sync' => SyncDriver::class,
        'roadrunner' => Queue::class,
    ],
    'registry' => [
        'handlers' => [],
        'serializers' => [
            WebhookHandler::class => 'symfony-json',
        ],
    ],
];
