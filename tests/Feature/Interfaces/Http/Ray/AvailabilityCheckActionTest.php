<?php

declare(strict_types=1);

namespace Tests\Feature\Interfaces\Http\Ray;

final class AvailabilityCheckActionTest extends RayTestCase
{
    public function testCheck(): void
    {
        $this->http->getJson('/_availability_check', headers: [
            'X-Buggregator-Event' => 'ray',
        ])
            ->assertStatus(400);
    }
}
