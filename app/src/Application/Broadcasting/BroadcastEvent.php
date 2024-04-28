<?php

declare(strict_types=1);

namespace App\Application\Broadcasting;

final readonly class BroadcastEvent
{
    public function __construct(
        public iterable|string|\Stringable $channel,
        public string|\Stringable $event,
        public array|\JsonSerializable $payload = [],
    ) {
    }
}
