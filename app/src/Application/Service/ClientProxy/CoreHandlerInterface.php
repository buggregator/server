<?php

declare(strict_types=1);

namespace App\Application\Service\ClientProxy;

use Buggregator\Client\Proto\Frame;

interface CoreHandlerInterface
{
    public function handle(Frame $frame): void;
}
