<?php

declare(strict_types=1);

namespace Modules\Events\Application;

use App\Application\Broadcasting\EventMapperRegistryInterface;
use Modules\Events\Application\Broadcasting\EventsWasClearMapper;
use Modules\Events\Application\Broadcasting\EventWasDeletedMapper;
use Modules\Events\Application\Broadcasting\EventWasReceivedMapper;
use Modules\Events\Domain\Events\EventsWasClear;
use Modules\Events\Domain\Events\EventWasDeleted;
use Modules\Events\Domain\Events\EventWasReceived;
use Spiral\Boot\Bootloader\Bootloader;

final class EventsBootloader extends Bootloader
{
    public function boot(
        EventMapperRegistryInterface $registry,
        EventWasReceivedMapper $eventWasReceivedMapper,
    ): void {
        $registry->register(
            event: EventWasDeleted::class,
            mapper: new EventWasDeletedMapper(),
        );

        $registry->register(
            event: EventsWasClear::class,
            mapper: new EventsWasClearMapper(),
        );

        $registry->register(
            event: EventWasReceived::class,
            mapper: $eventWasReceivedMapper,
        );
    }
}
