<?php

declare(strict_types=1);

namespace Modules\Ray\Interfaces\Http\Controllers\Locks;

use Psr\SimpleCache\CacheInterface;
use Spiral\Http\Request\InputManager;
use Spiral\Router\Annotation\Route;

final class DeleteAction
{
    #[Route(route: 'ray/locks/<hash>', name: 'ray.lock.delete', methods: 'POST', group: 'api')]
    public function __invoke(InputManager $request, CacheInterface $cache, string $hash): void
    {
        $cache->set($hash, [
            'active' => false,
            'stop_execution' => (bool) $request->input('stop_execution'),
        ]);
    }
}
