<?php

declare(strict_types=1);

namespace App\Application\Event;

interface EventTypeRegistryInterface
{
    public function register(string $type, EventTypeMapperInterface $mapper): void;
}
