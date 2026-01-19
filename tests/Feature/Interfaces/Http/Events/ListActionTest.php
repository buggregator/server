<?php

declare(strict_types=1);

namespace Tests\Feature\Interfaces\Http\Events;

use Database\Factory\EventFactory;
use Modules\Events\Domain\Event;
use Modules\Events\Domain\EventRepositoryInterface;
use Modules\Events\Domain\ValueObject\Timestamp;
use Modules\Projects\Domain\ValueObject\Key;
use Tests\Feature\Interfaces\Http\ControllerTestCase;

final class ListActionTest extends ControllerTestCase
{
    public function testCursorPagination(): void
    {
        $event1 = $this->createEventWithTimestamp('100');
        $event2 = $this->createEventWithTimestamp('200');
        $event3 = $this->createEventWithTimestamp('300');

        $response = $this->http
            ->listEvents(limit: 2)
            ->assertOk();

        $data = $response->json();

        $this->assertCount(2, $data['data']);
        $this->assertSame(2, $data['meta']['limit']);
        $this->assertTrue($data['meta']['has_more']);
        $this->assertNotEmpty($data['meta']['next_cursor']);

        $this->assertSame((string) $event3->getUuid(), $data['data'][0]['uuid']);
        $this->assertSame((string) $event2->getUuid(), $data['data'][1]['uuid']);

        $nextResponse = $this->http
            ->listEvents(limit: 2, cursor: $data['meta']['next_cursor'])
            ->assertOk();

        $nextData = $nextResponse->json();

        $this->assertCount(1, $nextData['data']);
        $this->assertSame(2, $nextData['meta']['limit']);
        $this->assertFalse($nextData['meta']['has_more']);
        $this->assertNull($nextData['meta']['next_cursor']);
        $this->assertSame((string) $event1->getUuid(), $nextData['data'][0]['uuid']);
    }

    public function testInvalidLimit(): void
    {
        $response = $this->http
            ->listEvents(limit: 0)
            ->assertUnprocessable()
            ->assertJsonResponseContains([
                'message' => 'The given data was invalid.',
                'code' => 422,
                'context' => 'Invalid pagination limit.',
            ]);

        $data = $response->json();

        $this->assertSame(['Limit must be a positive integer.'], $data['errors']['limit']);
    }

    private function createEventWithTimestamp(string $timestamp): Event
    {
        $event = EventFactory::new([
            'type' => 'foo',
            'project' => Key::create('default'),
            'timestamp' => Timestamp::typecast($timestamp),
        ])->makeOne();

        $this->get(EventRepositoryInterface::class)->store($event);

        return $event;
    }
}
