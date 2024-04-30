<?php

declare(strict_types=1);

namespace App\Application\Event;

final readonly class EventType
{
    public function __construct(
        public string $type,
        public ?string $project = null,
    ) {}
}
