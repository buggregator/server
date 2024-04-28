<?php

declare(strict_types=1);

namespace Modules\Events\Interfaces\Commands;

use App\Application\Commands\FindProjectByKey;
use App\Application\Commands\HandleReceivedEvent;
use App\Application\Domain\Entity\Json;
use Modules\Events\Domain\Event;
use Modules\Events\Domain\EventRepositoryInterface;
use Modules\Events\Domain\Events\EventWasReceived;
use Modules\Events\Domain\ValueObject\Timestamp;
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
        $project = null;
        // If the project is not null, we will find the project by key
        if ($command->project !== null) {
            $project = $this->queryBus->ask(new FindProjectByKey($command->project));
        }

        $this->events->store(
            $event = new Event(
                uuid: $command->uuid,
                type: $command->type,
                payload: new Json($command->payload),
                timestamp: Timestamp::create(),
                project: $project?->getKey(),
            ),
        );

        $this->dispatcher->dispatch(
            new EventWasReceived($event),
        );
    }
}
