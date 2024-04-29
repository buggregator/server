<?php

namespace Modules\Events\Interfaces\Queries;

use App\Application\Commands\AskEvents;

abstract class EventsHandler
{
    protected static function getScopeFromFindEvents(AskEvents $query): array
    {
        $scope = [];
        if ($query->type !== null) {
            $scope['type'] = $query->type;
        }

        $scope['project'] = $query->project;

        return $scope;
    }
}
