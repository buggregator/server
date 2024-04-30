<?php

declare(strict_types=1);

namespace Modules\Sentry\Application\Mapper;

use App\Application\Event\EventTypeMapperInterface;

final readonly class EventTypeMapper implements EventTypeMapperInterface
{
    public function __construct(
        public int $maxExceptions = 3,
    ) {}

    public function toPreview(string $type, array|\JsonSerializable $payload): array|\JsonSerializable
    {
        $data = $payload instanceof \JsonSerializable ? $payload->jsonSerialize() : $payload;

        return [
            'message' => $data['message'] ?? null,
            'exception' => $this->limitExceptionFramesNumber(
                exception: $data['exception'] ?? null,
                max: $this->maxExceptions,
            ),
            'level' => $data['level'] ?? null,
            'platform' => $data['platform'],
            'environment' => $data['environment'],
            'server_name' => $data['server_name'],
            'event_id' => $data['event_id'] ?? null,
        ];
    }

    public function limitExceptionFramesNumber(array|null $exception, int $max = 3): array|null
    {
        if ($exception === null) {
            return null;
        }

        foreach ($exception['values'] as $i => $e) {
            if (!isset($e['stacktrace']['frames'])) {
                continue;
            }

            $exception['values'][$i]['stacktrace']['frames'] = \array_slice($e['stacktrace']['frames'], -$max);
        }

        return $exception;
    }
}
