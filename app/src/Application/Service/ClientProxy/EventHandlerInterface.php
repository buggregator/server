<?php

declare(strict_types=1);

namespace App\Application\Service\ClientProxy;

interface EventHandlerInterface
{
    public function isSupported(string $type): bool;
    public function handle(string $payload): void;
}
