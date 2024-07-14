<?php

declare(strict_types=1);

namespace Modules\Events\Interfaces\Commands;

use App\Application\Commands\DeleteEvent;
use Modules\Events\Domain\EventRepositoryInterface;
use Modules\Events\Domain\Events\EventWasDeleted;
use Psr\EventDispatcher\EventDispatcherInterface;
use Spiral\Cqrs\Attribute\CommandHandler;

final readonly class DeleteEventHandler
{
    public function __construct(
        private EventRepositoryInterface $events,
        private EventDispatcherInterface $dispatcher,
    ) {}

    #[CommandHandler]
    public function __invoke(DeleteEvent $command): void
    {
        $event = $this->events->findByPK((string) $command->uuid);
        if ($event === null) {
            return;
        }

        if ($this->events->deleteByPK((string) $command->uuid)) {
            $this->dispatcher->dispatch(
                new EventWasDeleted(
                    uuid: $command->uuid,
                    project: $event->getProject(),
                ),
            );
        }
    }
}
