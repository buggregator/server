<?php

declare(strict_types=1);

namespace App\Application\Service\ClientProxy;

interface CoreHandlerInterface
{
    public function handle(string $type, string $payload): void;
}
