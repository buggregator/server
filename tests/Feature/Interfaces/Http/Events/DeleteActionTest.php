<?php

declare(strict_types=1);

namespace Tests\Feature\Interfaces\Http\Events;

use Tests\Feature\Interfaces\Http\ControllerTestCase;

final class DeleteActionTest extends ControllerTestCase
{
    public function testDeleteEvent(): void
    {
        $event = $this->createEvent();

        $this->http
            ->deleteEvent($event->getUuid())
            ->assertSuccessResource();

        $this->assertEventMissing($event);
    }

    public function testDeleteNonExistEvent(): void
    {
        $this->http
            ->deleteEvent($this->randomUuid())
            ->assertSuccessResource();
    }
}
