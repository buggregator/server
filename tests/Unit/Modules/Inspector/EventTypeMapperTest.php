<?php

declare(strict_types=1);

namespace Tests\Unit\Modules\Inspector;

use Tests\DatabaseTestCase;

final class EventTypeMapperTest extends DatabaseTestCase
{
    public function testMapEvent(): void
    {
        $event = $this->createEvent(type: 'inspector');
        $data = $event->getPayload()->jsonSerialize();

        $this->assertSame([
            $data[0],
        ], $this->mapEventTypeToPreview($event));
    }
}
