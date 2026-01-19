<?php

declare(strict_types=1);

namespace Modules\Events\Interfaces\Queries;

use App\Application\Commands\CountEventsByType;
use Modules\Events\Domain\EventRepositoryInterface;
use Spiral\Cqrs\Attribute\QueryHandler;
use Spiral\Cqrs\QueryBusInterface;

final class CountEventsByTypeHandler extends EventsHandler
{
    public function __construct(
        private readonly EventRepositoryInterface $events,
        QueryBusInterface $bus,
    ) {
        parent::__construct($bus);
    }

    #[QueryHandler]
    public function __invoke(CountEventsByType $query): array
    {
        return $this->events->countByType($this->getScopeFromFindEvents($query));
    }
}
