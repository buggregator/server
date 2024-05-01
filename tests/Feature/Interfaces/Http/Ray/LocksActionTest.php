<?php

declare(strict_types=1);

namespace Tests\Feature\Interfaces\Http\Ray;

use Psr\SimpleCache\CacheInterface;

final class LocksActionTest extends RayTestCase
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
