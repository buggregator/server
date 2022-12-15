<?php

declare(strict_types=1);

namespace Modules\Profiler\Application\Handlers;

use Modules\Profiler\Application\EventHandlerInterface;

final class CleanupEvent implements EventHandlerInterface
{

    public function handle(array $event): array
    {
        unset($event['profile']);

        return $event;
    }
}
