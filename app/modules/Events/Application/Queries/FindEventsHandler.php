<?php

declare(strict_types=1);

namespace Modules\Events\Application\Queries;

use App\Application\Commands\FindEvents;
use Modules\Events\Domain\EventRepositoryInterface;
use Spiral\Cqrs\Attribute\QueryHandler;

final class FindEventsHandler extends EventsHandler
{
    public function __construct(
        private readonly EventRepositoryInterface $events
    ) {
    }

    #[QueryHandler]
    public function __invoke(FindEvents $query): iterable
    {
        return $this->events->findAll(
            self::getScopeFromFindEvents($query),
            ['date' => 'desc']
        );
    }
}
