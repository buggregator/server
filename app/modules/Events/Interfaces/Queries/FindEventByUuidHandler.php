<?php

declare(strict_types=1);

namespace Modules\Events\Interfaces\Queries;

use App\Application\Commands\FindEventByUuid;
use App\Application\Exception\EntityNotFoundException;
use Modules\Events\Domain\Event;
use Modules\Events\Domain\EventRepositoryInterface;
use Spiral\Cqrs\Attribute\QueryHandler;

final class FindEventByUuidHandler
{
    public function __construct(
        private readonly EventRepositoryInterface $events
    ) {
    }

    #[QueryHandler]
    public function __invoke(FindEventByUuid $query): Event
    {
        $event = $this->events->findByPK((string)$query->uuid);
        if (!$event) {
            throw new EntityNotFoundException(
                \sprintf('Event with given uuid [%s] was not found.', (string)$query->uuid)
            );
        }

        return $event;
    }
}
