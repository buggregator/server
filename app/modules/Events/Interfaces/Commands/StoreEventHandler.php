<?php

declare(strict_types=1);

namespace Modules\Events\Interfaces\Commands;

use App\Application\Commands\HandleReceivedEvent;
use App\Application\Domain\Entity\Json;
use Modules\Events\Domain\Event;
use Modules\Events\Domain\EventRepositoryInterface;
use Modules\Events\Domain\Events\EventWasReceived;
use Psr\EventDispatcher\EventDispatcherInterface;
use Spiral\Cqrs\Attribute\CommandHandler;
use Spiral\Cqrs\QueryBusInterface;

final readonly class StoreEventHandler
{
    public function __construct(
        private EventDispatcherInterface $dispatcher,
        private EventRepositoryInterface $events,
        private QueryBusInterface $queryBus,
    ) {
    }

    #[CommandHandler]
    public function handle(HandleReceivedEvent $command): void
    {
        $this->events->store(
            new Event(
                uuid: $command->uuid,
                type: $command->type,
                payload: new Json($command->payload),
                timestamp: $command->timestamp,
                project: $command->project,
            ),
        );

        $this->dispatcher->dispatch(
            new EventWasReceived(
                uuid: $command->uuid,
                type: $command->type,
                payload: $command->payload,
                timestamp: $command->timestamp,
                project: $command->project,
            ),
        );
    }
}
