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
        'profiles' => [
            'adapter' => 'local',
            'directory' => directory('runtime') . 'profiles',
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
            'prefix' => 'smtp',
        ],
        'http_dumps' => [
            'server' => 'attachments',
            'prefix' => 'http_dumps',
        ],
        'profiles' => [
            'server' => 'profiles',
        ],
    ],
];
