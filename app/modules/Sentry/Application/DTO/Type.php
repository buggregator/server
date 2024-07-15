<?php

declare(strict_types=1);

namespace Modules\Sentry\Application\DTO;

enum Type
{
    case Event;
    case CheckIn;
    case Transaction;
    case ReplyEvent;
    case ReplayRecording;
    case Span;
}
