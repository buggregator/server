<?php

declare(strict_types=1);

use Spiral\Broadcasting\Driver\LogBroadcast;
use Spiral\Broadcasting\Driver\NullBroadcast;
use Spiral\Security\Actor\Actor;

return [
    'default' => env('BROADCAST_CONNECTION', 'null'),
    'aliases' => [],
    'connections' => [
        'centrifugo' => [
            'driver' => 'centrifugo',
        ],
        'null' => [
            'driver' => NullBroadcast::class,
        ],
    ],
];
