<?php

declare(strict_types=1);

namespace Modules\Events\Interfaces\Commands;

use App\Application\Commands\ClearEvents;
use Modules\Events\Domain\EventRepositoryInterface;
use Modules\Events\Domain\Events\EventsWasClear;
use Psr\EventDispatcher\EventDispatcherInterface;
use Spiral\Cqrs\Attribute\CommandHandler;

final class ClearEventsHandler
{
    public function __construct(
        private readonly EventRepositoryInterface $events,
        private readonly EventDispatcherInterface $dispatcher,
    ) {
    }

    #[CommandHandler]
    public function __invoke(ClearEvents $command): void
    {
        $args = [];
        if ($command->type) {
            $args['type'] = $command->type;
        }

        if ($command->uuids) {
            $args['uuid'] = $command->uuids;
        }

        $this->events->deleteAll($args);
        $this->dispatcher->dispatch(new EventsWasClear(type: $command->type));
    }
}
