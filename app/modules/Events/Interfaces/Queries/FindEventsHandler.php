<?php

declare(strict_types=1);

namespace Modules\Events\Interfaces\Queries;

use App\Application\Commands\FindEvents;
use Modules\Events\Domain\EventRepositoryInterface;
use Spiral\Cqrs\Attribute\QueryHandler;

final class FindEventsHandler extends EventsHandler
{
    public function __construct(
        private readonly EventRepositoryInterface $events,
    ) {
    }

    #[QueryHandler]
    public function __invoke(FindEvents $query): iterable
    {
        return $this->events->findAll(
            scope: self::getScopeFromFindEvents($query),
            orderBy: ['timestamp' => 'desc'],
            limit: $query->limit,
        );
    }
}
