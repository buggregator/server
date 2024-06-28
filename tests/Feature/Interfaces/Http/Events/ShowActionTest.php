<?php

declare(strict_types=1);

namespace Tests\Feature\Interfaces\Http\Events;

use Modules\Events\Interfaces\Http\Resources\EventResource;
use Tests\Feature\Interfaces\Http\ControllerTestCase;

final class ShowActionTest extends ControllerTestCase
{
    public function testShowEvent(): void
    {
        $event = $this->createEvent();

        $this->http
            ->showEvent($event->getUuid())
            ->assertOk()
            ->assertResource(
                new EventResource($event),
            );
    }

    public function testNotFoundShowEvent(): void
    {
        $uuid = $this->randomUuid();

        $this->http
            ->showEvent($uuid)
            ->assertNotFound()
            ->assertJsonResponseSame([
                'message' => 'Event with given uuid [' . $uuid . '] was not found.',
                'code' => 404,
            ]);
    }
}
