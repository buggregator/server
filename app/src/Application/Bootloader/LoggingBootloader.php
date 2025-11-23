<?php

declare(strict_types=1);

namespace App\Application\Bootloader;

use Spiral\Boot\Bootloader\Bootloader;

final class LoggingBootloader extends Bootloader
{
    public function defineDependencies(): array
    {
        return [
            LoggingBootloader::class,
        ];
    }
}
