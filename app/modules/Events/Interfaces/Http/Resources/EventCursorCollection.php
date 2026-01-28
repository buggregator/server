<?php

declare(strict_types=1);

namespace Modules\Events\Interfaces\Http\Resources;

use App\Application\HTTP\Response\ResourceCollection;

final class EventCursorCollection extends ResourceCollection
{
    public function __construct(
        iterable $data,
        private readonly int $limit,
        private readonly bool $hasMore,
        private readonly ?string $nextCursor,
    ) {
        parent::__construct(
            $data,
            EventResource::class,
        );
    }

    protected function wrapData(array $data, array $meta = []): array
    {
        $meta = [
            'limit' => $this->limit,
            'has_more' => $this->hasMore,
            'next_cursor' => $this->nextCursor,
        ];

        return parent::wrapData($data, $meta);
    }
}
