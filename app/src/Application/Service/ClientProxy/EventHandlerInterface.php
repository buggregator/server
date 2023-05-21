<?php

declare(strict_types=1);

namespace App\Application\Service\ClientProxy;

use Buggregator\Client\Proto\Frame;

interface EventHandlerInterface
{
    public function isSupported(Frame $frame): bool;
    public function handle(Frame $frame): void;
}
