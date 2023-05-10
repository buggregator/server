<?php

declare(strict_types=1);

namespace Modules\Ray\Interfaces\Http\Controllers\Locks;

use Psr\SimpleCache\CacheInterface;
use Spiral\Http\Exception\ClientException\NotFoundException;
use Spiral\Router\Annotation\Route;

final class CheckAction
{
    #[Route(route: '/locks/<hash>', name: 'ray.lock.check')]
    public function __invoke(CacheInterface $cache, string $hash): array
    {
        $lock = $cache->get($hash);

        if (!$lock) {
            throw new NotFoundException();
        }

        if (\is_array($lock)) {
            $cache->delete($hash);

            return $lock;
        }

        return ['active' => true, 'stop_execution' => false];
    }
}
