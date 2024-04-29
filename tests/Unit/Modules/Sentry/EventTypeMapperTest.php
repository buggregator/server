<?php

declare(strict_types=1);

namespace Tests\Unit\Modules\Sentry;

use Tests\DatabaseTestCase;

final class EventTypeMapperTest extends DatabaseTestCase
{
    public function testMapEvent(): void
    {
        $event = $this->createEvent(type: 'sentry');
        $data = $event->getPayload()->jsonSerialize();

        $this->assertSame([
            'message' => null,
            'exception' => $data['exception'],
            'level' => null,
            'platform' => $data['platform'],
            'environment' => $data['environment'],
            'server_name' => $data['server_name'],
            'event_id' => $data['event_id'] ?? null,
        ], $this->mapEventTypeToPreview($event));
    }
}
