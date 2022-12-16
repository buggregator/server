<?php

declare(strict_types=1);

use App\Application\TCP\ExceptionHandlerInterceptor;
use Modules\VarDumper\Interfaces\TCP\Service as VarDumperService;
use Modules\Monolog\Interfaces\TCP\Service as MonologService;
use Modules\Smtp\Interfaces\TCP\Service as SmtpService;

return [
    'services' => [
        'var-dumper' => VarDumperService::class,
        'monolog' => MonologService::class,
        'smtp' => SmtpService::class,
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
