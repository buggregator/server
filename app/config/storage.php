<?php

declare(strict_types=1);

return [
    'default' => env('STORAGE_SERVER', 'attachments'),

    'servers' => [
        'attachments' => [
            'adapter' => 'local',
            'directory' => directory('runtime') . 'attachments',
            'visibility' => [
                'public' => ['file' => 0644, 'dir' => 0755],
                'private' => ['file' => 0600, 'dir' => 0700],

                'default' => 'public',
            ],
        ],
    ],

    'buckets' => [
        'smtp' => [
            'server' => 'attachments',
            'prefix' => 'attachments',
        ],
        'http_dumps' => [
            'server' => 'attachments',
            'prefix' => 'attachments',
        ],
    ],
];
