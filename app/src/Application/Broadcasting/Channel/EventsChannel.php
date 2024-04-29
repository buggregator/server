<?php

declare(strict_types=1);

namespace App\Application\Broadcasting\Channel;

final class EventsChannel extends Channel
{
    public function __construct(string|\Stringable|null $project = null)
    {
        if ($project !== null) {
            parent::__construct('events.project.' . $project);
        } else {
            parent::__construct('events');
        }
    }
}
