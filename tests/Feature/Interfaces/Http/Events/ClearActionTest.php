<?php

declare(strict_types=1);

namespace Tests\Feature\Interfaces\Http\Events;

use Tests\Feature\Interfaces\Http\ControllerTestCase;

final class ClearActionTest extends ControllerTestCase
{
    public function testClearAllEvents(): void
    {
        $this->fakeEvents()->eventShouldBeClear();

        $this->http
            ->clearEvents()
            ->assertSuccessResource();
    }

    public function testClearEventsByType(): void
    {
        $this->fakeEvents()->eventShouldBeClear(type: 'test');

        $this->http
            ->clearEvents(type: 'test')
            ->assertSuccessResource();
    }
}
