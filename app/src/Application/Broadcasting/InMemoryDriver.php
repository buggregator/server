<?php

declare(strict_types=1);

namespace App\Application\Broadcasting;

use Spiral\Broadcasting\Driver\AbstractBroadcast;
use Spiral\Core\Attribute\Singleton;

#[Singleton]
final class InMemoryDriver extends AbstractBroadcast
{
    private static array $published = [];

    public function publish(iterable|\Stringable|string $topics, iterable|string $messages): void
    {
        /** @var non-empty-string[] $topics */
        $topics = $this->formatTopics($this->toArray($topics));

        foreach ($topics as $topic) {
            foreach ($this->toArray($messages) as $message) {
                \assert(\is_string($message));
                self::$published[$topic][] = \json_decode($message, true);
            }
        }
    }

    public function published(): array
    {
        return self::$published;
    }

    public function reset(): void
    {
        self::$published = [];
    }
}
