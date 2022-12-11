<?php

declare(strict_types=1);

namespace Modules\Events\Domain\Events;

use App\Application\Broadcasting\Channel\EventsChannel;
use App\Application\Broadcasting\ShouldBroadcastInterface;
use App\Application\Domain\ValueObjects\Uuid;

class EventWasDeleted implements ShouldBroadcastInterface
{
    public function __construct(
        public readonly Uuid $uuid
    ) {
    }

    public function jsonSerialize(): array
    {
        return [
            'uuid' => (string)$this->uuid,
        ];
    }

    public function getEventName(): string
    {
        return 'event.deleted';
    }

    public function getBroadcastTopics(): iterable|string|\Stringable
    {
        return new EventsChannel();
    }
}
