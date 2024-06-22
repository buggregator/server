<?php

declare(strict_types=1);

namespace App\Application\Event;

enum StackStrategy
{
    case OnlyLatest;
    case All;
    case None;
}
