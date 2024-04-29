<?php

declare(strict_types=1);

namespace Modules\Events\Application\Broadcasting;

use App\Application\Broadcasting\BroadcastEvent;
use App\Application\Broadcasting\Channel\EventsChannel;
use App\Application\Broadcasting\EventMapperInterface;
use Modules\Events\Domain\Events\EventsWasClear;

final readonly class EventsWasClearMapper implements EventMapperInterface
{
    /**
     * @param EventsWasClear $event
     */
    public function toBroadcast(object $event): BroadcastEvent
    {
        return new BroadcastEvent(
            channel: new EventsChannel($event->project),
            event: 'events.clear',
            payload: [
                'type' => $event->type,
                'project' => $event->project,
            ],
        );
    }
}
