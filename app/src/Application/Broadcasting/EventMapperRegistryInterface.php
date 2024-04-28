<?php

declare(strict_types=1);

namespace App\Application\Broadcasting;

interface EventMapperRegistryInterface
{
    /**
     * @param class-string $event
     */
    public function register(string $event, EventMapperInterface $mapper): void;
}
