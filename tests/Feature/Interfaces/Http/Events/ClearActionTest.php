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

    public function testClearEventsByUuids(): void
    {
        $this->fakeEvents()->eventShouldBeClear(uuids: ['foo', 'bar']);

        $this->http
            ->clearEvents(uuids: ['foo', 'bar'])
            ->assertSuccessResource();
    }

    public function testClearEventsByTypeAndUuids(): void
    {
        $this->fakeEvents()->eventShouldBeClear(type: 'test', uuids: ['foo', 'bar']);

        $this->http
            ->clearEvents(type: 'test', uuids: ['foo', 'bar'])
            ->assertSuccessResource();
    }
}
