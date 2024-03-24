<?php

declare(strict_types=1);

namespace Modules\Events\Domain\Events;

use App\Application\Broadcasting\Channel\EventsChannel;
use App\Application\Broadcasting\ShouldBroadcastInterface;
use App\Application\Domain\ValueObjects\Uuid;

final class EventWasReceived implements ShouldBroadcastInterface
{
    public function __construct(
        public readonly Uuid $uuid,
        public readonly string $type,
        public readonly array $payload,
        public readonly float $timestamp,
        public readonly ?int $projectId = null,
    ) {
    }

    public function jsonSerialize(): array
    {
        return [
            'project_id' => $this->projectId,
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
