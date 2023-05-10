<?php

declare(strict_types=1);

namespace Modules\Ray\Application;

enum TypeEnum: string
{
    case CreateLock = 'create_lock';
    case ClearAll = 'clear_all';
}
