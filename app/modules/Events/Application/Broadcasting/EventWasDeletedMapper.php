<?php

declare(strict_types=1);

namespace Modules\Events\Application\Broadcasting;

use App\Application\Broadcasting\BroadcastEvent;
use App\Application\Broadcasting\Channel\EventsChannel;
use App\Application\Broadcasting\EventMapperInterface;
use Modules\Events\Domain\Events\EventWasDeleted;

final readonly class EventWasDeletedMapper implements EventMapperInterface
{
    /**
     * @param EventWasDeleted $event
     */
    public function toBroadcast(object $event): BroadcastEvent
    {
        return new BroadcastEvent(
            channel: new EventsChannel($event->project),
            event: 'event.deleted',
            payload: [
                'uuid' => (string)$event->uuid,
                'project' => $event->project,
            ],
        );
    }
}
