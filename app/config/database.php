<?php

declare(strict_types=1);

use Cycle\Database\Config\PostgresDriverConfig;
use Cycle\Database\Config\Postgres\TcpConnectionConfig as PgTcpConnectionConfig;
use Cycle\Database\Config\Mysql\TcpConnectionConfig as MysqlTcpConnectionConfig;
use Cycle\Database\Config\MySQLDriverConfig;
use Cycle\Database\Config\SQLiteDriverConfig;
use Cycle\Database\Config\SQLite\FileConnectionConfig;

return [
    'logger' => [
        'default' => env('DB_LOGGER'),
        'drivers' => [],
    ],

    'default' => 'default',

    'databases' => [
        'default' => [
            'driver' => env('DB_DRIVER', 'mysql'),
        ],
    ],

    'drivers' => [
        'pgsql' => new PostgresDriverConfig(
            connection: new PgTcpConnectionConfig(
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
        'mysql' => new MySQLDriverConfig(
            connection: new MysqlTcpConnectionConfig(
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
        'sqlite' => new SQLiteDriverConfig(
            connection: new FileConnectionConfig(database: directory('runtime') . 'test.db'),
            options: [
                'logQueryParameters' => env('DB_LOG_QUERY_PARAMETERS', false),
            ],
        ),
    ],
];
