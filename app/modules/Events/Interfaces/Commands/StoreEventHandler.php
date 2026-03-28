<?php

declare(strict_types=1);

namespace Modules\Events\Interfaces\Commands;

use Spiral\Cqrs\CommandBusInterface;
use App\Application\Commands\CreateProject;
use App\Application\Commands\FindProjectByKey;
use App\Application\Commands\HandleReceivedEvent;
use App\Application\Domain\ValueObjects\Json;
use Modules\Events\Application\EventMetrics;
use Modules\Events\Domain\Event;
use Modules\Events\Domain\EventRepositoryInterface;
use Modules\Events\Domain\Events\EventWasReceived;
use Modules\Events\Domain\ValueObject\Timestamp;
use Modules\Projects\Domain\Project;
use Modules\Projects\Domain\ValueObject\Key;
use Psr\EventDispatcher\EventDispatcherInterface;
use Spiral\Cqrs\Attribute\CommandHandler;
use Spiral\Cqrs\QueryBusInterface;

final readonly class StoreEventHandler
{
    public function __construct(
        private EventDispatcherInterface $dispatcher,
        private EventRepositoryInterface $events,
        private QueryBusInterface $queryBus,
        private CommandBusInterface $commandBus,
        private EventMetrics $metrics,
    ) {}

    #[CommandHandler]
    public function handle(HandleReceivedEvent $command): void
    {
        $projectKey = $command->project ?? Project::DEFAULT_KEY;

        $project = $this->queryBus->ask(new FindProjectByKey($projectKey));

        if ($project === null) {
            try {
                $project = $this->commandBus->dispatch(
                    new CreateProject(key: $projectKey, name: $projectKey),
                );
            } catch (\Throwable) {
                // Race condition: project was created between check and create
                $project = $this->queryBus->ask(new FindProjectByKey($projectKey));
            }
        }

        $this->events->store(
            $event = new Event(
                uuid: $command->uuid,
                type: $command->type,
                payload: new Json($command->payload),
                timestamp: Timestamp::create(),
                project: $project?->getKey() ?? Key::create(Project::DEFAULT_KEY),
            ),
        );

        $this->dispatcher->dispatch(
            new EventWasReceived($event),
        );

        $this->metrics->newEvent(type: $event->getType());
    }
}
