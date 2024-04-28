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
                'name' => '/foo',
                'type' => 'process',
                'hash' => '979ffbcecfcd3a72c1c4d536aa1fe85b3e996df1d7093755b9b4a1ded8e33b5c',
                'host' =>
                    [
                        'hostname' => 'ButschsterLpp',
                        'ip' => '127.0.1.1',
                        'os' => 'Linux',
                    ],
                'timestamp' => 1701464039.650622,
                'memory_peak' => 15.53,
                'duration' => 0.22,
            ],
            [
                'model' => 'segment',
                'type' => 'my-process',
                'host' =>
                    [
                        'hostname' => 'ButschsterLpp',
                        'ip' => '127.0.1.1',
                        'os' => 'Linux',
                    ],
                'transaction' =>
                    [
                        'name' => '/foo',
                        'hash' => '979ffbcecfcd3a72c1c4d536aa1fe85b3e996df1d7093755b9b4a1ded8e33b5c',
                        'timestamp' => 1701464039.650622,
                    ],
                'start' => 0.2,
                'timestamp' => 1701464039.650826,
                'duration' => 0.01,
            ],
        ];
    }
}
