<?php

declare(strict_types=1);

namespace Modules\Events\Interfaces\Queries;

final readonly class EventsCursorResult
{
    /**
     * @param array $items
     */
    public function __construct(
        public array $items,
        public int $limit,
        public bool $hasMore,
        public ?string $nextCursor,
    ) {}
}
