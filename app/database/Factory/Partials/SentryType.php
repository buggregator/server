<?php

declare(strict_types=1);

namespace Database\Factory\Partials;

trait SentryType
{
    protected static function getSentryPayload(): array
    {
        return [
            'timestamp' => 1701455435.634665,
            'platform' => 'php',
            'sdk' => [
                'name' => 'sentry.php',
                'version' => '4.0.1',
            ],
            'server_name' => 'Localhost',
            'environment' => 'production',
            'modules' => [
                'buggregator/app' => 'v1.0.0',
                'cycle/orm' => 'v2.5.0',
                'spiral/framework' => '3.10.0',
            ],
            'contexts' => [
                'os' => [
                    'name' => 'Linux',
                    'version' => '5.15.133.1-microsoft-standard-WSL2',
                    'build' => '#1 SMP Thu Oct 5 21:02:42 UTC 2023',
                    'kernel_version' => 'Linux Test x86_64',
                ],
                'runtime' => [
                    'name' => 'php',
                    'version' => '8.2.5',
                ],
                'trace' => [
                    'trace_id' => '143ef743ce184eb7abd0ae0891d33b7d',
                    'span_id' => 'e4a276672c8a4a38',
                ],
            ],
            'exception' => [
                'values' => [
                    [
                        'type' => 'Exception',
                        'value' => 'test',
                        'stacktrace' => [
                            'frames' => [
                                [
                                    'filename' => '/vendor/phpunit/phpunit/phpunit',
                                    'lineno' => 107,
                                    'in_app' => true,
                                    'abs_path' => '\\/vendor/phpunit/phpunit/phpunit',
                                    'pre_context' => [
                                        0 => '',
                                        1 => 'unset($options);',
                                        2 => '',
                                        3 => 'require PHPUNIT_COMPOSER_INSTALL;',
                                        4 => '',
                                    ],
                                    'context_line' => 'PHPUnit\\TextUI\\Command::main();',
                                    'post_context' => ['',],
                                ],
                                [
                                    'filename' => '/vendor/phpunit/phpunit/src/TextUI/Command.php',
                                    'lineno' => 97,
                                    'in_app' => true,
                                    'abs_path' => '\\/vendor/phpunit/phpunit/src/TextUI/Command.php',
                                    'function' => 'PHPUnit\\TextUI\\Command::main',
                                    'raw_function' => 'PHPUnit\\TextUI\\Command::main',
                                    'pre_context' => [
                                        0 => '     * @throws Exception',
                                        1 => '     */',
                                        2 => '    public static function main(bool $exit = true): int',
                                        3 => '    {',
                                        4 => '        try {',
                                    ],
                                    'context_line' => '            return (new static)->run($_SERVER[\'argv\'], $exit);',
                                    'post_context' => [
                                        0 => '        } catch (Throwable $t) {',
                                        1 => '            throw new RuntimeException(',
                                        2 => '                $t->getMessage(),',
                                        3 => '                (int) $t->getCode(),',
                                        4 => '                $t,',
                                    ],
                                ],
                                [
                                    'filename' => '/vendor/phpunit/phpunit/src/TextUI/Command.php',
                                    'lineno' => 144,
                                    'in_app' => true,
                                    'abs_path' => '\\/vendor/phpunit/phpunit/src/TextUI/Command.php',
                                    'function' => 'PHPUnit\\TextUI\\Command::run',
                                    'raw_function' => 'PHPUnit\\TextUI\\Command::run',
                                    'pre_context' => [
                                        0 => '        }',
                                        1 => '',
                                        2 => '        unset($this->arguments[\'test\'], $this->arguments[\'testFile\']);',
                                        3 => '',
                                        4 => '        try {',
                                    ],
                                    'context_line' => '            $result = $runner->run($suite, $this->arguments, $this->warnings, $exit);',
                                    'post_context' => [
                                        0 => '        } catch (Throwable $t) {',
                                        1 => '            print $t->getMessage() . PHP_EOL;',
                                        2 => '        }',
                                        3 => '',
                                        4 => '        $return = TestRunner::FAILURE_EXIT;',
                                    ],
                                ],
                            ],
                        ],
                        'mechanism' => [
                            'type' => 'generic',
                            'handled' => true,
                            'data' => [
                                'code' => 0,
                            ],
                        ],
                    ],
                ],
            ],
        ];
    }
}
