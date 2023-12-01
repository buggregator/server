<?php

declare(strict_types=1);

namespace Modules\Events\Interfaces\Commands;

use App\Application\Commands\DeleteEvent;
use Modules\Events\Domain\EventRepositoryInterface;
use Modules\Events\Domain\Events\EventWasDeleted;
use Psr\EventDispatcher\EventDispatcherInterface;
use Spiral\Cqrs\Attribute\CommandHandler;

final class DeleteEventHandler
{
    public function __construct(
        private readonly EventRepositoryInterface $events,
        private readonly EventDispatcherInterface $dispatcher
    ) {
    }

    #[CommandHandler]
    public function __invoke(DeleteEvent $command): void
    {
        if ($this->events->deleteByPK((string)$command->uuid)) {
            $this->dispatcher->dispatch(new EventWasDeleted($command->uuid));
        }
    }
}
