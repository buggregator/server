<?php

declare(strict_types=1);

use Monolog\Handler\ErrorLogHandler;
use Monolog\Handler\SyslogHandler;
use Monolog\Logger;
use Monolog\Processor\PsrLogMessageProcessor;

return [
    'default' => env('MONOLOG_DEFAULT_CHANNEL', 'roadrunner'),
    'globalLevel' => Logger::toMonologLevel(env('MONOLOG_DEFAULT_LEVEL', Logger::DEBUG)),
    'handlers' => [
        'roadrunner' => [
            \Spiral\RoadRunnerBridge\Logger\Handler::class,
        ],
    ],
    'processors' => [],
];
