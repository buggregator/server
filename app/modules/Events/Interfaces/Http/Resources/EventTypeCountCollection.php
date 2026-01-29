<?php

declare(strict_types=1);

namespace Modules\Events\Interfaces\Http\Resources;

use App\Application\HTTP\Response\ResourceCollection;

final class EventTypeCountCollection extends ResourceCollection
{
    public function __construct(
        iterable $data,
    ) {
        parent::__construct(
            $data,
            EventTypeCountResource::class,
        );
    }
}
