<?php

declare(strict_types=1);

namespace Modules\Events\Application\Commands;

use App\Application\Commands\FindProjectByName;
use App\Application\Commands\HandleReceivedEvent;
use App\Application\Domain\Entity\Json;
use Carbon\Carbon;
use Modules\Events\Domain\Event;
use Modules\Events\Domain\EventRepositoryInterface;
use Modules\Events\Domain\Events\EventWasReceived;
use Psr\EventDispatcher\EventDispatcherInterface;
use Spiral\Cqrs\Attribute\CommandHandler;
use Spiral\Cqrs\QueryBusInterface;

final class StoreEventHandler
{
    public function __construct(
        private readonly EventDispatcherInterface $dispatcher,
        private readonly EventRepositoryInterface $events,
        private readonly QueryBusInterface $queryBus
    ) {
    }

    #[CommandHandler]
    public function handle(HandleReceivedEvent $command): void
    {
        $projectId = null;
        if ($command->project !== null) {
            $projectId = $this->queryBus->ask(new FindProjectByName($command->project));
        }

        $this->events->store(
            new Event(
                $command->uuid,
                $command->type,
                new Json($command->payload),
                Carbon::createFromTimestamp($command->timestamp)->toDateTimeImmutable(),
                $projectId,
            )
        );

        $this->dispatcher->dispatch(
            new EventWasReceived(
                uuid: $command->uuid,
                type: $command->type,
                payload: $command->payload,
                timestamp: $command->timestamp,
                sendToConsole: $command->sendToConsole,
                projectId: $projectId,
            )
        );
    }
}
