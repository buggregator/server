<?php

declare(strict_types=1);

use Spiral\Queue\Driver\SyncDriver;
use Spiral\RoadRunner\Jobs\Queue\MemoryCreateInfo;
use Spiral\RoadRunner\Jobs\Queue\AMQPCreateInfo;
use Spiral\RoadRunner\Jobs\Queue\BeanstalkCreateInfo;
use Spiral\RoadRunner\Jobs\Queue\SQSCreateInfo;
use Spiral\RoadRunnerBridge\Queue\Queue;

return [
    'default' => env('QUEUE_CONNECTION', 'sync'),
    'aliases' => [],
    'connections' => [
        'sync' => [
            'driver' => 'sync',
        ],
        'roadrunner' => [
            'driver' => 'roadrunner',
            'default' => 'memory',
            'pipelines' => [
                'memory' => [
                    'connector' => new MemoryCreateInfo('local'),
                    // Run consumer for this pipeline on startup (by default)
                    // You can pause consumer for this pipeline via console command
                    // php app.php queue:pause local
                    'consume' => true,
                ]
            ],
        ],
    ],

    'driverAliases' => [
        'sync' => SyncDriver::class,
        'roadrunner' => Queue::class,
    ],

    'registry' => [
        'handlers' => [],
    ],
];
