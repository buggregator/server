<?php

declare(strict_types=1);

namespace App\Application\Broadcasting;

interface EventMapperInterface
{
    public function toBroadcast(object $event): BroadcastEvent;
}
