<?php

declare(strict_types=1);

namespace Modules\Sentry\Application;

use App\Application\Event\EventType;
use Modules\Sentry\Application\DTO\Payload;

interface EventHandlerInterface
{
    public function handle(Payload $payload, EventType $event): Payload;
}
