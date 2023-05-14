<?php

declare(strict_types=1);

namespace App\Application\Service\HttpHandler;

interface HandlerRegistryInterface
{
    public function register(HandlerInterface $handler): void;
}
