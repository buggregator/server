<?php

declare(strict_types=1);

namespace Database\Factory\Partials;

trait InspectorType
{
    protected static function getInspectorPayload(): array
    {
        return [
            [
                'model' => 'transaction',
                'name' => 'http://127.0.0.1:8080/blog/post/1',
                'type' => 'request',
                'hash' => 'a78992af325ea2335d102726ad8bc323a8ffcb5e4a004df9633f988a24c6fd4c',
                'host' => [
                    'hostname' => 'ButschsterLpp',
                    'ip' => '127.0.1.1',
                    'os' => 'Linux',
                ],
                'http' => [
                    'request' => [
                        'method' => 'GET',
                        'version' => 'unknown',
                        'socket' => [
                            'remote_address' => '127.0.0.1',
                        ],
                    ],
                    'url' => [
                        'protocol' => 'http',
                        'path' => 'app.php',
                        'search' => '?',
                    ],
                ],
                'timestamp' => 1714336189.047214,
                'memory_peak' => 17.08,
                'duration' => 2.73,
            ],
            [
                'model' => 'segment',
                'type' => 'App\\\\Middleware\\\\InspectorMiddleware',
                'host' => [
                    'hostname' => 'ButschsterLpp',
                    'ip' => '127.0.1.1',
                    'os' => 'Linux',
                ],
                'transaction' => [
                    'name' => 'http://127.0.0.1:8080/dump1',
                    'hash' => 'a78992af325ea2335d102726ad8bc323a8ffcb5e4a004df9633f988a24c6fd4c',
                    'timestamp' => 1714336189.047214,
                ],
                'start' => 0.07,
                'timestamp' => 1714336189.047291,
                'duration' => 1.49,
            ],
            [
                'model' => 'segment',
                'type' => 'exception',
                'label' => 'Test exception',
                'host' => [
                    'hostname' => 'ButschsterLpp',
                    'ip' => '127.0.1.1',
                    'os' => 'Linux',
                ],
                'transaction' => [
                    'name' => 'http://127.0.0.1:8080/dump1',
                    'hash' => 'a78992af325ea2335d102726ad8bc323a8ffcb5e4a004df9633f988a24c6fd4c',
                    'timestamp' => 1714336189.047214,
                ],
                'start' => 1.65,
                'timestamp' => 1714336189.048868,
                'context' => [
                    'Error' => [
                        'model' => 'error',
                        'timestamp' => 1714336189.048884,
                        'host' => [
                            'hostname' => 'ButschsterLpp',
                            'ip' => '127.0.1.1',
                            'os' => 'Linux',
                        ],
                        'message' => 'Test exception',
                        'class' => 'Exception',
                        'file' => '/root/repos/spiral-apps/filters-bridge/app/src/Middleware/InspectorMiddleware.php',
                        'line' => 32,
                        'code' => 0,
                        'stack' => [
                            [
                                'class' => 'App\\Middleware\\InspectorMiddleware',
                                'function' => 'process',
                                'args' => [],
                                'type' => '->',
                                'file' => '/root/repos/spiral-apps/filters-bridge/app/src/Middleware/InspectorMiddleware.php',
                                'line' => 32,
                                'code' => [
                                    [
                                        'line' => 29,
                                        'code' => '        $response = $this->inspector->addSegment(function ($segment) use ($handler, $request) {',
                                    ],
                                    [
                                        'line' => 30,
                                        'code' => '            return $handler->handle($request);',
                                    ],
                                    [
                                        'line' => 31,
                                        'code' => '        }, self::class);',
                                    ],
                                    [
                                        'line' => 32,
                                        'code' => '',
                                    ],
                                    [
                                        'line' => 33,
                                        'code' => '        $this->inspector->reportException(new \\Exception(\'Test exception\'));',
                                    ],
                                    [
                                        'line' => 34,
                                        'code' => '',
                                    ],
                                    [
                                        'line' => 35,
                                        'code' => '        $this->inspector->flush();',
                                    ],
                                    [
                                        'line' => 36,
                                        'code' => '',
                                    ],
                                    [
                                        'line' => 37,
                                        'code' => '        dump($this->inspector);',
                                    ],
                                    [
                                        'line' => 38,
                                        'code' => '',
                                    ],
                                ],
                            ],
                            [
                                'class' => 'Spiral\\Http\\Pipeline',
                                'function' => 'Spiral\\Http\\{closure}',
                                'args' => [],
                                'type' => '->',
                                'file' => '[internal]',
                                'line' => '0',
                                'code' => [],
                            ],
                            [
                                'class' => 'ReflectionFunction',
                                'function' => 'invokeArgs',
                                'args' => [],
                                'type' => '->',
                                'file' => '/root/repos/spiral-apps/filters-bridge/vendor/spiral/framework/src/Core/src/Internal/Invoker.php',
                                'line' => 74,
                                'code' => [
                                    [
                                        'line' => 71,
                                        'code' => '                throw new ContainerException($e->getMessage(), $e->getCode(), $e);',
                                    ],
                                    [
                                        'line' => 72,
                                        'code' => '            }',
                                    ],
                                    [
                                        'line' => 73,
                                        'code' => '',
                                    ],
                                    [
                                        'line' => 74,
                                        'code' => '            // Invoking Closure with resolved arguments',
                                    ],
                                    [
                                        'line' => 75,
                                        'code' => '            return $reflection->invokeArgs(',
                                    ],
                                    [
                                        'line' => 76,
                                        'code' => '                $this->resolver->resolveArguments($reflection, $parameters)',
                                    ],
                                    [
                                        'line' => 77,
                                        'code' => '            );',
                                    ],
                                    [
                                        'line' => 78,
                                        'code' => '        }',
                                    ],
                                    [
                                        'line' => 79,
                                        'code' => '',
                                    ],
                                    [
                                        'line' => 80,
                                        'code' => '        throw new NotCallableException(\'Unsupported callable.\');',
                                    ],
                                ],
                            ],
                        ],
                        'transaction' => [
                            'name' => 'http://127.0.0.1:8080/dump1',
                            'hash' => 'a78992af325ea2335d102726ad8bc323a8ffcb5e4a004df9633f988a24c6fd4c',
                        ],
                        'handled' => true,
                    ],
                ],
                'duration' => 1.05,
            ],
            [
                'model' => 'error',
                'timestamp' => 1714336189.048884,
                'host' => [
                    'hostname' => 'ButschsterLpp',
                    'ip' => '127.0.1.1',
                    'os' => 'Linux',
                ],
                'message' => 'Test exception',
                'class' => 'Exception',
                'file' => '/root/repos/spiral-apps/filters-bridge/app/src/Middleware/InspectorMiddleware.php',
                'line' => 32,
                'code' => 0,
                'stack' => [
                    [
                        'class' => 'App\\Middleware\\InspectorMiddleware',
                        'function' => 'process',
                        'args' =>
                            [
                            ],
                        'type' => '->',
                        'file' => '/root/repos/spiral-apps/filters-bridge/app/src/Middleware/InspectorMiddleware.php',
                        'line' => 32,
                        'code' => [
                            [
                                'line' => 29,
                                'code' => '        $response = $this->inspector->addSegment(function ($segment) use ($handler, $request) {',
                            ],
                            [
                                'line' => 30,
                                'code' => '            return $handler->handle($request);',
                            ],
                            [
                                'line' => 31,
                                'code' => '        }, self::class);',
                            ],
                            [
                                'line' => 32,
                                'code' => '',
                            ],
                            [
                                'line' => 33,
                                'code' => '        $this->inspector->reportException(new \\Exception(\'Test exception\'));',
                            ],
                            [
                                'line' => 34,
                                'code' => '',
                            ],
                            [
                                'line' => 35,
                                'code' => '        $this->inspector->flush();',
                            ],
                            [
                                'line' => 36,
                                'code' => '',
                            ],
                            [
                                'line' => 37,
                                'code' => '        dump($this->inspector);',
                            ],
                            [
                                'line' => 38,
                                'code' => '',
                            ],
                        ],
                    ],
                ],
                'transaction' => [
                    'name' => 'http://127.0.0.1:8080/blog/post/1',
                    'hash' => 'a78992af325ea2335d102726ad8bc323a8ffcb5e4a004df9633f988a24c6fd4c',
                ],
                'handled' => true,
            ],
        ];
    }
}
