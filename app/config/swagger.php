<?php

return [
    'documentation' => [
        'info' => [
            'title' => 'Buggregator API',
            'description' => '',
            'version' => env('API_VERSION', '1.0.0'),
        ],
        'components' => [
            'schemas' => [
                'ResponseMeta' => [
                    'type' => 'object',
                    'properties' => [
                        'grid' => [
                            'type' => 'object',
                            'properties' => [
                                // TODO: add grid meta
                            ],
                        ],
                    ],
                ],
                'NotFoundError' => [
                    'type' => 'object',
                    'properties' => [
                        'error' => [
                            'type' => 'string',
                            'example' => 'Http Error - 404',
                        ],
                        'status' => [
                            'type' => 'integer',
                            'example' => 404,
                        ],
                    ],
                ],
                'ValidationError' => [
                    'type' => 'object',
                    'properties' => [
                        'message' => [
                            'type' => 'string',
                            'example' => 'The given data was invalid.',
                        ],
                        'code' => [
                            'type' => 'integer',
                            'example' => 433,
                        ],
                        'errors' => [
                            'type' => 'object',
                            'properties' => [
                                'field' => [
                                    'type' => 'string',
                                ],
                            ],
                        ],
                        'context' => [
                            'type' => 'object',
                        ],
                    ],
                ],
            ],
        ],
    ],
    'paths' => [
        directory('app') . 'src/Application/HTTP/Response',
        directory('app') . 'src/Interfaces/Http',
        directory('app') . 'modules/Events/Interfaces/Http',
        directory('app') . 'modules/Projects/Interfaces/Http',
        directory('app') . 'modules/Webhooks/Interfaces/Http',
        directory('app') . 'modules/Smtp/Interfaces/Http',
        directory('app') . 'modules/HttpDumps/Interfaces/Http',
    ],
];
