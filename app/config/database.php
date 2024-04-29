<?php

declare(strict_types=1);

use Cycle\Database\Config;

return [
    'logger' => [
        'default' => env('DB_LOGGER'),
        'drivers' => [],
    ],

    'default' => 'default',

    'databases' => [
        'default' => [
            'driver' => env('DB_DRIVER', 'pgsql'),
        ],
    ],

    'drivers' => [
        'pgsql' => new Config\PostgresDriverConfig(
            connection: new Config\Postgres\TcpConnectionConfig(
                database: env('DB_DATABASE', 'buggregator'),
                host: env('DB_HOST', '127.0.0.1'),
                port: env('DB_PORT', 5432),
                user: env('DB_USERNAME', 'postgres'),
                password: env('DB_PASSWORD'),
            ),
            schema: 'public',
            queryCache: true,
            options: [
                'withDatetimeMicroseconds' => true,
                'logQueryParameters' => env('DB_LOG_QUERY_PARAMETERS', false),
            ],
        ),
        'mysql' => new Config\MySQLDriverConfig(
            connection: new Config\MySQL\TcpConnectionConfig(
                database: env('DB_DATABASE', 'buggregator'),
                host: env('DB_HOST', '127.0.0.1'),
                port: env('DB_PORT', 3306),
                user: env('DB_USERNAME', 'root'),
                password: env('DB_PASSWORD'),
            ),
            queryCache: true,
            options: [
                'withDatetimeMicroseconds' => true,
                'logQueryParameters' => env('DB_LOG_QUERY_PARAMETERS', false),
            ],
        ),

        // Only for testing purposes
        // SQLite does not support multiple connections in the same time
        'sqlite' => new Config\SQLiteDriverConfig(
            connection: new Config\SQLite\MemoryConnectionConfig(),
            options: [
                'logQueryParameters' => env('DB_LOG_QUERY_PARAMETERS', false),
            ],
        ),
    ],
];
