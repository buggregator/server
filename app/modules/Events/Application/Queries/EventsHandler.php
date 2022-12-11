<?php

namespace Modules\Events\Application\Queries;

use App\Application\Commands\AskEvents;

abstract class EventsHandler
{
    protected static function getScopeFromFindEvents(AskEvents $query): array
    {
        $scope = [];
        if ($query->type !== null) {
            $scope['type'] = $query->type;
        }
        if ($query->projectId !== null) {
            $scope['project_id'] = $query->projectId;
        }

        return $scope;
    }
}
