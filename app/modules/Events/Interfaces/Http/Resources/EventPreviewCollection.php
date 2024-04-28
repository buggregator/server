<?php

declare(strict_types=1);

namespace Modules\Events\Interfaces\Http\Resources;

use App\Application\Event\EventTypeMapperInterface;
use App\Application\HTTP\Response\ResourceCollection;
use Modules\Events\Domain\Event;

final class EventPreviewCollection extends ResourceCollection
{
    public function __construct(
        iterable $data,
        EventTypeMapperInterface $mapper,
    ) {
        parent::__construct(
            $data,
            static fn(Event $event) => new EventPreviewResource($event, $mapper),
        );
    }
}
