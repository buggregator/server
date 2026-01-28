<?php

declare(strict_types=1);

namespace Modules\Events\Interfaces\Http\Resources;

use App\Application\Event\EventTypeMapperInterface;
use App\Application\HTTP\Response\ResourceCollection;
use Modules\Events\Domain\Event;

final class EventPreviewCursorCollection extends ResourceCollection
{
    public function __construct(
        iterable $data,
        EventTypeMapperInterface $mapper,
        private readonly int $limit,
        private readonly bool $hasMore,
        private readonly ?string $nextCursor,
    ) {
        parent::__construct(
            $data,
            static fn(Event $event) => new EventPreviewResource($event, $mapper),
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
