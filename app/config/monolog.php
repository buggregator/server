<?php

declare(strict_types=1);
use Spiral\RoadRunnerBridge\Logger\Handler;
use Monolog\Logger;

return [
    'default' => env('MONOLOG_DEFAULT_CHANNEL', 'roadrunner'),
    'globalLevel' => Logger::toMonologLevel(env('MONOLOG_DEFAULT_LEVEL', Logger::DEBUG)),
    'handlers' => [
        'roadrunner' => [
            Handler::class,
        ],
    ],
    'processors' => [],
];
