<?php

namespace Modules\Events\Application\Queries;

use App\Application\Commands\AskEvents;

abstract class EventsHandler
{
    protected static function getScopeFromFindEvents(AskEvents $query): array
    {
        $scope = [];
        $scope['type'] = $query->type;
        if ($query->projectId) {
            $scope['project_id'] = $query->projectId;
        }

        return $scope;
    }
}
