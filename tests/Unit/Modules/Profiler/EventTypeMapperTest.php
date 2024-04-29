<?php

declare(strict_types=1);

namespace Tests\Unit\Modules\Profiler;

use Tests\DatabaseTestCase;

final class EventTypeMapperTest extends DatabaseTestCase
{
    public function testMapEvent(): void
    {
        $event = $this->createEvent(type: 'profiler');
        $data = $event->getPayload()->jsonSerialize();

        $this->assertSame([
            'peaks' => $data['peaks'],
            'tags' => $data['tags'],
            'app_name' => $data['app_name'],
            'hostname' => $data['hostname'],
            'date' => $data['date'],
        ], $this->mapEventTypeToPreview($event));
    }
}
