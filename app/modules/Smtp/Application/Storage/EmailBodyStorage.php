<?php

declare(strict_types=1);

namespace Modules\Smtp\Application\Storage;

use Carbon\Carbon;
use Psr\SimpleCache\CacheInterface;

final readonly class EmailBodyStorage
{
    public function __construct(
        private CacheInterface $cache,
    ) {
    }

    public function getMessage(string $uuid): Message
    {
        return $this->cache->get($this->getCacheKey($uuid), new Message($uuid));
    }

    public function persist(Message $message): void
    {
        $this->cache->set(
            $this->getCacheKey($message->uuid),
            $message,
            Carbon::now()->addMinutes(1)->diffAsCarbonInterval(),
        );
    }

    public function delete(Message $message): void
    {
        $this->cache->delete($this->getCacheKey($message->uuid));
    }

    private function getCacheKey(string $uuid): string
    {
        return $uuid;
    }
}
