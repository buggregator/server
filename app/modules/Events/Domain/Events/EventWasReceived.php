<?php

declare(strict_types=1);

namespace Modules\Events\Domain\Events;

use App\Application\Broadcasting\ShouldBroadcastInterface;
use Modules\Events\Domain\Event;

final readonly class EventWasReceived implements ShouldBroadcastInterface
{
    public function __construct(
        public Event $event,
    ) {
    }
}
