<?php

declare(strict_types=1);

namespace Modules\Profiler\Application;

interface EventHandlerInterface
{
    public function handle(array $event): array;
}
