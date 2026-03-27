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
            'platform' => $data['platform'] ?? null,
            'environment' => $data['environment'] ?? null,
            'server_name' => $data['server_name'] ?? null,
            'event_id' => $data['event_id'] ?? null,
        ];
    }

    public function toSearchableText(string $type, array|\JsonSerializable $payload): string
    {
        $data = $payload instanceof \JsonSerializable ? $payload->jsonSerialize() : $payload;

        $parts = \array_filter([
            $data['message'] ?? null,
            $data['level'] ?? null,
            $data['environment'] ?? null,
            $data['server_name'] ?? null,
            $data['platform'] ?? null,
        ]);

        foreach (($data['exception']['values'] ?? []) as $e) {
            if (isset($e['type'])) {
                $parts[] = $e['type'];
            }
            if (isset($e['value'])) {
                $parts[] = $e['value'];
            }
        }

        return \implode(' ', $parts);
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
