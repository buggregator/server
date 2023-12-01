<?php

declare(strict_types=1);

namespace Interfaces\Http\Ray;

use Tests\Feature\Interfaces\Http\ControllerTestCase;

final class AvailabilityCheckActionTest extends ControllerTestCase
{
    public function testCheck(): void
    {
        $this->http->getJson('/_availability_check', headers: [
            'X-Buggregator-Event' => 'ray',
        ])
            ->assertStatus(400);
    }
}
