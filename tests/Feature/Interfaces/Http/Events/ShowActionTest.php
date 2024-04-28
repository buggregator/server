<?php

declare(strict_types=1);

namespace Tests\Feature\Interfaces\Http\Events;

use App\Application\Domain\Entity\Json;
use App\Application\Event\EventTypeMapperInterface;
use Modules\Events\Domain\Event;
use Modules\Events\Interfaces\Http\Resources\EventResource;
use Tests\Feature\Interfaces\Http\ControllerTestCase;

final class ShowActionTest extends ControllerTestCase
{
    public function testShowEvent(): void
    {
        $event = new Event(
            uuid: $this->randomUuid(),
            type: 'test',
            payload: new Json(['foo' => 'bar']),
            timestamp: 123.456,
            project: null,
        );

        $this->fakeEvents()
            ->shouldRequestEventByUuid($event->getUuid())
            ->andReturnEvent($event);

        $this->http
            ->showEvent($event->getUuid())
            ->assertOk()
            ->assertResource(
                new EventResource(
                    $event,
                    $this->get(EventTypeMapperInterface::class),
                ),
            );
    }

    public function testNotFoundShowEvent(): void
    {
        $uuid = $this->randomUuid();
        $this->fakeEvents()
            ->shouldRequestEventByUuid($uuid)
            ->andThrowNotFound();

        $this->http
            ->showEvent($uuid)
            ->assertNotFound()
            ->assertJsonResponseSame([
                'message' => 'Event not found',
                'code' => 404,
            ]);
    }
}
