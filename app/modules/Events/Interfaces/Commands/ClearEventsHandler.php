<?php

declare(strict_types=1);

namespace Modules\Events\Interfaces\Commands;

use App\Application\Commands\ClearEvents;
use Modules\Events\Domain\EventRepositoryInterface;
use Modules\Events\Domain\Events\EventsWasClear;
use Modules\Projects\Domain\Project;
use Modules\Projects\Domain\ValueObject\Key;
use Psr\EventDispatcher\EventDispatcherInterface;
use Spiral\Cqrs\Attribute\CommandHandler;

final readonly class ClearEventsHandler
{
    public function __construct(
        private EventRepositoryInterface $events,
        private EventDispatcherInterface $dispatcher,
    ) {}

    #[CommandHandler]
    public function __invoke(ClearEvents $command): void
    {
        $scope = ['project' => $command->project ?? (string) Key::create(Project::DEFAULT_KEY)];

        if ($command->type) {
            $scope['type'] = $command->type;
        }

        if ($command->uuids) {
            $scope['uuid'] = $command->uuids;
        }

        $this->events->deleteAll($scope);
        $this->dispatcher->dispatch(
            new EventsWasClear(
                type: $command->type,
                project: $command->project,
            ),
        );
    }
}
