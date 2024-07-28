<?php

namespace Modules\Events\Interfaces\Queries;

use App\Application\Commands\AskEvents;
use App\Application\Commands\FindAllProjects;
use Modules\Projects\Domain\ProjectInterface;
use Modules\Projects\Domain\ValueObject\Key;
use Spiral\Cqrs\QueryBusInterface;

abstract class EventsHandler
{
    public function __construct(
        private readonly QueryBusInterface $bus,
    ) {}

    protected function getScopeFromFindEvents(AskEvents $query): array
    {
        $scope = [];
        if ($query->type !== null) {
            $scope['type'] = $query->type;
        }

        if ($query->project !== null) {
            $scope['project'] = $query->project;
        } elseif ($query->project !== []) {
            // TODO: refactor this
            $projects = $this->bus->ask(new FindAllProjects());
            $keys = \array_map(
                static fn(ProjectInterface $project): Key => $project->getKey(),
                \iterator_to_array($projects),
            );

            $scope['project'] = $keys;
        }

        return $scope;
    }
}
