<?php

declare(strict_types=1);

namespace Modules\Monolog\Application;

use Modules\Monolog\Application\Mapper\EventTypeMapper;
use App\Application\Event\EventTypeRegistryInterface;
use Spiral\Boot\Bootloader\Bootloader;

final class MonologEventBootloader extends Bootloader
{
    public function boot(EventTypeRegistryInterface $registry): void
    {
        $registry->register('monolog', new EventTypeMapper());
    }
}
