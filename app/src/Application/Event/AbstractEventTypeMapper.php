<?php

declare(strict_types=1);

namespace App\Application\Event;

abstract readonly class AbstractEventTypeMapper implements EventTypeMapperInterface
{
    public function toPreview(string $type, array|\JsonSerializable $payload): array|\JsonSerializable
    {
        return $payload;
    }

    public function toFull(string $type, array|\JsonSerializable $payload): array|\JsonSerializable
    {
        return $payload;
    }
}
