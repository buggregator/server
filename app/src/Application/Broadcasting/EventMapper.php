<?php

declare(strict_types=1);

namespace App\Application\Broadcasting;

use Spiral\Core\Attribute\Singleton;

#[Singleton]
final class EventMapper implements EventMapperInterface, EventMapperRegistryInterface
{
    /** @var array<class-string, EventMapperInterface> */
    private array $mappers = [];

    public function register(string $event, EventMapperInterface $mapper): void
    {
        $this->mappers[$event] = $mapper;
    }

    public function toBroadcast(object $event): BroadcastEvent
    {
        foreach ($this->mappers as $eventClass => $mapper) {
            if ($event instanceof $eventClass) {
                return $mapper->toBroadcast($event);
            }
        }

        throw new \RuntimeException('No mapper found for event ' . $event::class);
    }
}
