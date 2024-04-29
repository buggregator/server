<?php

declare(strict_types=1);

namespace App\Application\Event;

interface EventTypeMapperInterface
{
    public function toPreview(string $type, array|\JsonSerializable $payload): array|\JsonSerializable;
}
