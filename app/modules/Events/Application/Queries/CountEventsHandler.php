<?php

declare(strict_types=1);

namespace Modules\Events\Application\Queries;

use App\Application\Commands\CountEvents;
use Modules\Events\Domain\EventRepositoryInterface;
use Spiral\Cqrs\Attribute\QueryHandler;

final class CountEventsHandler extends EventsHandler
{
    public function __construct(
        private readonly EventRepositoryInterface $events
    ) {
    }

    #[QueryHandler]
    public function __invoke(CountEvents $query): int
    {
        return $this->events->countAll(self::getScopeFromFindEvents($query));
    }
}
