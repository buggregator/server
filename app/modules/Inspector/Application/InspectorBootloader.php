<?php

declare(strict_types=1);

namespace Modules\Inspector\Application;

use App\Application\Service\HttpHandler\HandlerRegistryInterface;
use Modules\Inspector\Interfaces\Http\Handler\EventHandler;
use Spiral\Boot\Bootloader\Bootloader;

final class InspectorBootloader extends Bootloader
{
    public function boot(
        HandlerRegistryInterface $registry,
        EventHandler $handler
    ): void {
        $registry->register($handler);
    }
}
