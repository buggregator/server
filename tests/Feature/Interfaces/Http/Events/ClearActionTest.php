<?php

declare(strict_types=1);

namespace Tests\Feature\Interfaces\Http\Events;

use Tests\Feature\Interfaces\Http\ControllerTestCase;

final class ClearActionTest extends ControllerTestCase
{
    public function testClearAllEvents(): void
    {
        $event1 = $this->createEvent(type: 'foo', project: 'default');
        $event2 = $this->createEvent(type: 'foo', project: 'text');
        $event3 = $this->createEvent(type: 'bar', project: 'default');
        $event4 = $this->createEvent(type: 'baz', project: 'text');

        $this->http
            ->clearEvents()
            ->assertSuccessResource();

        $this
            ->assertEventExists($event2, $event4)
            ->assertEventMissing($event1, $event3);
    }

    public function testClearEventsByType(): void
    {
        $event1 = $this->createEvent(type: 'foo', project: 'default');
        $event2 = $this->createEvent(type: 'foo', project: 'default');
        $event3 = $this->createEvent(type: 'bar', project: 'default');

        $this->http
            ->clearEvents(type: 'bar')
            ->assertSuccessResource();

        $this
            ->assertEventExists($event1, $event2)
            ->assertEventMissing($event3);
    }

    public function testClearEventsByProject(): void
    {
        $event1 = $this->createEvent(type: 'foo', project: 'test');
        $event2 = $this->createEvent(type: 'foo', project: 'default');
        $event3 = $this->createEvent(type: 'bar', project: 'test');
        $event4 = $this->createEvent(type: 'baz', project: 'default');

        $this->http
            ->clearEvents(project: 'default')
            ->assertSuccessResource();

        $this
            ->assertEventMissing($event2, $event4)
            ->assertEventExists($event1, $event3);
    }

    public function testClearEventsByUuids(): void
    {
        $event1 = $this->createEvent(type: 'foo', project: 'default');
        $event2 = $this->createEvent(type: 'foo', project: 'default');
        $event3 = $this->createEvent(type: 'foo', project: 'default');
        $event4 = $this->createEvent(type: 'foo', project: 'default');
        $event5 = $this->createEvent(type: 'bar', project: 'test');

        $this->http
            ->clearEvents(uuids: [
                (string) $event1->getUuid(),
                (string) $event2->getUuid(),
                (string) $event3->getUuid(),
                (string) $event4->getUuid(),
            ])
            ->assertSuccessResource();

        $this
            ->assertEventMissing($event1, $event2, $event3, $event4)
            ->assertEventExists($event5);
    }

    public function testClearEventsByTypeAndUuids(): void
    {
        $event1 = $this->createEvent(type: 'foo', project: 'default');
        $event2 = $this->createEvent(type: 'foo', project: 'default');
        $event3 = $this->createEvent(type: 'foo', project: 'default');
        $event4 = $this->createEvent(type: 'foo', project: 'default');
        $event5 = $this->createEvent(type: 'bar', project: 'default');

        $this->http
            ->clearEvents(type: 'foo', uuids: [
                (string) $event1->getUuid(),
                (string) $event2->getUuid(),
                (string) $event3->getUuid(),
            ])
            ->assertSuccessResource();

        $this
            ->assertEventMissing($event1, $event2, $event3)
            ->assertEventExists($event4, $event5);
    }
}
