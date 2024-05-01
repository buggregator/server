<?php

declare(strict_types=1);

namespace Tests\Unit\Modules\VarDumper;

use Tests\DatabaseTestCase;

final class EventTypeMapperTest extends DatabaseTestCase
{
    public function testMapEvent(): void
    {
        $event = $this->createEvent(type: 'var-dump');
        $this->markTestIncomplete('This test has not been implemented yet.');
    }
}
