<?php

declare(strict_types=1);

namespace Modules\Events\Domain\Events;

use App\Application\Broadcasting\Channel\EventsChannel;
use App\Application\Broadcasting\ShouldBroadcastInterface;

final readonly class EventsWasClear implements ShouldBroadcastInterface
{
    public function __construct(
        public ?string $type,
        public ?string $project = null,
    ) {
    }

    public function jsonSerialize(): array
    {
        return [
            'type' => $this->type,
            'project' => $this->project,
        ];
    }

    public function getEventName(): string
    {
        return 'event.cleared';
    }

    public function getBroadcastTopics(): iterable|string|\Stringable
    {
        return new EventsChannel($this->project);
    }
}
