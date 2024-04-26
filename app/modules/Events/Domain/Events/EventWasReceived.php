<?php

declare(strict_types=1);

namespace Modules\Events\Domain\Events;

use App\Application\Broadcasting\Channel\EventsChannel;
use App\Application\Broadcasting\ShouldBroadcastInterface;
use App\Application\Domain\ValueObjects\Uuid;

final readonly class EventWasReceived implements ShouldBroadcastInterface
{
    public function __construct(
        public Uuid $uuid,
        public string $type,
        public array $payload,
        public float $timestamp,
        public ?string $project = null,
    ) {
    }

    public function jsonSerialize(): array
    {
        return [
            'project' => $this->project,
            'uuid' => (string)$this->uuid,
            'type' => $this->type,
            'payload' => $this->payload,
            'timestamp' => $this->timestamp,
        ];
    }

    public function getEventName(): string
    {
        return 'event.received';
    }

    public function getBroadcastTopics(): iterable|string|\Stringable
    {
        return new EventsChannel();
    }
}
