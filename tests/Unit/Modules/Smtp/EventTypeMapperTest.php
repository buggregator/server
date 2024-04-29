<?php

declare(strict_types=1);

namespace Tests\Unit\Modules\Smtp;

use Tests\DatabaseTestCase;

final class EventTypeMapperTest extends DatabaseTestCase
{
    public function testMapEvent(): void
    {
        $event = $this->createEvent(type: 'smtp');
        $data = $event->getPayload()->jsonSerialize();

        $this->assertSame([
            'subject' => $data['subject'],
            'from' => $data['from'],
            'to' => $data['to'],
        ], $this->mapEventTypeToPreview($event));
    }
}
