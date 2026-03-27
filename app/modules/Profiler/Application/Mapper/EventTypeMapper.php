<?php

declare(strict_types=1);

namespace Modules\Profiler\Application\Mapper;

use App\Application\Event\EventTypeMapperInterface;

final readonly class EventTypeMapper implements EventTypeMapperInterface
{
    public function toPreview(string $type, array|\JsonSerializable $payload): array|\JsonSerializable
    {
        $data = $payload instanceof \JsonSerializable ? $payload->jsonSerialize() : $payload;

        return [
            'peaks' => $data['peaks'],
            'tags' => $data['tags'],
            'app_name' => $data['app_name'],
            'hostname' => $data['hostname'],
            'date' => $data['date'],
        ];
    }

    public function toSearchableText(string $type, array|\JsonSerializable $payload): string
    {
        $data = $payload instanceof \JsonSerializable ? $payload->jsonSerialize() : $payload;

        $parts = \array_filter([
            $data['app_name'] ?? null,
            $data['hostname'] ?? null,
        ]);

        foreach (($data['tags'] ?? []) as $key => $value) {
            $parts[] = \is_string($key) ? "{$key}:{$value}" : $value;
        }

        return \implode(' ', $parts);
    }
}
