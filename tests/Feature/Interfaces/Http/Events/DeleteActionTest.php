<?php

declare(strict_types=1);

namespace Tests\Feature\Interfaces\Http\Events;

use Tests\Feature\Interfaces\Http\ControllerTestCase;

final class DeleteActionTest extends ControllerTestCase
{
    public function testDeleteEvent(): void
    {
        $event = $this->createEvent();

        $events = $this->fakeEvents();
        $events->eventShouldBeFound($event->getUuid(), $event);
        $events->eventShouldBeDeleted($event->getUuid());

        $this->http
            ->deleteEvent($event->getUuid())
            ->assertSuccessResource();
    }

    public function testDeleteNonExistEvent(): void
    {
        $event = $this->createEvent();

        $events = $this->fakeEvents();
        $events->eventShouldBeFound($event->getUuid(), null);
        $events->eventShouldNotBeDeleted($event->getUuid());

        $this->http
            ->deleteEvent($event->getUuid())
            ->assertSuccessResource();
    }
}
