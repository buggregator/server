<?php

declare(strict_types=1);

namespace App\Application\Broadcasting;

use Stringable;

interface ShouldBroadcastInterface extends \JsonSerializable
{
    public function getEventName(): string;

    public function getBroadcastTopics(): iterable|string|Stringable;
}
