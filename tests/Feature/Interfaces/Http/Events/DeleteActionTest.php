<?php

declare(strict_types=1);

namespace Tests\Feature\Interfaces\Http\Events;

use Tests\Feature\Interfaces\Http\ControllerTestCase;

final class DeleteActionTest extends ControllerTestCase
{
    public function testDeleteEvent(): void
    {
        $uuid = $this->randomUuid();

        $this->fakeEvents()->eventShouldBeDeleted($uuid);

        $this->http
            ->deleteEvent($uuid)
            ->assertSuccessResource();
    }
}
