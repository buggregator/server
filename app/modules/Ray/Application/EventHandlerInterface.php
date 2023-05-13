<?php

declare(strict_types=1);

namespace Modules\Ray\Application;

interface EventHandlerInterface
{
    public function handle(array $event): array;
}
