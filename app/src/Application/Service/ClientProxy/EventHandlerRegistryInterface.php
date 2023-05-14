<?php

declare(strict_types=1);

namespace App\Application\Service\ClientProxy;

interface EventHandlerRegistryInterface
{
    public function register(EventHandlerInterface $handler): void;
}
