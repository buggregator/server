<?php

declare(strict_types=1);

namespace Modules\Events\Application\Broadcasting;

use App\Application\Broadcasting\BroadcastEvent;
use App\Application\Broadcasting\Channel\EventsChannel;
use App\Application\Broadcasting\EventMapperInterface;
use App\Application\Event\EventTypeMapperInterface;
use Modules\Events\Domain\Events\EventWasReceived;

final readonly class EventWasReceivedMapper implements EventMapperInterface
{
    public function __construct(
        private EventTypeMapperInterface $eventTypeMapper,
    ) {}

    /**
     * @param EventWasReceived $event
     */
    public function toBroadcast(object $event): BroadcastEvent
    {
        return new BroadcastEvent(
            channel: new EventsChannel($event->event->getProject()),
            event: 'event.received',
            payload: [
                'uuid' => (string) $event->event->getUuid(),
                'project' => $event->event->getProject(),
                'type' => $event->event->getType(),
                'payload' => $this->eventTypeMapper->toPreview(
                    type: $event->event->getType(),
                    payload: $event->event->getPayload(),
                ),
                'timestamp' => $event->event->getTimestamp(),
            ],
        );
    }
}
