<?php

declare(strict_types=1);

namespace Modules\Events\Domain\Events;

use App\Application\Broadcasting\ShouldBroadcastInterface;

final readonly class EventsWasClear implements ShouldBroadcastInterface
{
    public function __construct(
        public ?string $type,
        public ?string $project = null,
    ) {
    }
}
