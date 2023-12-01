<?php

declare(strict_types=1);

namespace Interfaces\Http\Ray;

use Psr\SimpleCache\CacheInterface;
use Tests\Feature\Interfaces\Http\ControllerTestCase;

final class LocksActionTest extends ControllerTestCase
{
    public function testCheck(): void
    {
        $this->http->getJson('/locks/123', headers: [
            'X-Buggregator-Event' => 'ray',
        ])
            ->assertOk()
            ->assertJsonResponseSame([
                'active' => true,
                'stop_execution' => false,
            ]);
    }

    public function testCheckWithLock(): void
    {
        $cache = $this->get(CacheInterface::class);
        $cache->set('123', $response = ['active' => false, 'stop_execution' => true]);

        $this->http->getJson('/locks/123', headers: [
            'X-Buggregator-Event' => 'ray',
        ])
            ->assertOk()
            ->assertJsonResponseSame($response);
    }
}
