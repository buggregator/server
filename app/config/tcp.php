<?php

declare(strict_types=1);

use App\Application\TCP\ExceptionHandlerInterceptor;
use Modules\Monolog\Interfaces\TCP\Service as MonologService;

return [
    'services' => [
        'monolog' => MonologService::class,
    ],

    'interceptors' => [
        'var-dumper' => [
            ExceptionHandlerInterceptor::class,
        ],
        'monolog' => [
            ExceptionHandlerInterceptor::class,
        ],
    ],

    'debug' => env('TCP_DEBUG', false),
];
