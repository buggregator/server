<?php

declare(strict_types=1);

namespace Database\Factory\Partials;

trait MonologType
{
    protected static function getMonologPayload(): array
    {
        return [
            'message' => 'Some message',
            'context' => [
                'project' => 'default',
            ],
            'level' => 400,
            'level_name' => 'ERROR',
            'channel' => 'socket',
            'datetime' => '2024-04-28T06:53:07.674031+00:00',
            'extra' => [],
        ];
    }
}
