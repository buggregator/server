<?php

declare(strict_types=1);

namespace Modules\Events\Domain\Events;

use App\Application\Broadcasting\Channel\EventChannel;
use App\Application\Broadcasting\ShouldBroadcastInterface;

class EventsWasClear implements ShouldBroadcastInterface
{
    public function __construct(
        public readonly ?string $type
    ) {
    }

    public function jsonSerialize(): array
    {
        return [
            'type' => $this->type,
        ];
    }

    public function getEventName(): string
    {
        return 'event.cleared';
    }

    public function getBroadcastTopics(): iterable|string|\Stringable
    {
        return new EventChannel();
    }
}
