<?php

declare(strict_types=1);

namespace Modules\Monolog\Application\Mapper;

use App\Application\Event\EventTypeMapperInterface;

final readonly class EventTypeMapper implements EventTypeMapperInterface
{
    public function toPreview(string $type, array|\JsonSerializable $payload): array|\JsonSerializable
    {
        return $payload;
    }

    public function toSearchableText(string $type, array|\JsonSerializable $payload): string
    {
        $data = $payload instanceof \JsonSerializable ? $payload->jsonSerialize() : $payload;

        return \implode(' ', \array_filter([
            $data['message'] ?? null,
            $data['channel'] ?? null,
            $data['level_name'] ?? null,
        ]));
    }
}
