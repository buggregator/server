<?php

declare(strict_types=1);

namespace Modules\Webhooks\Interfaces\Http\Resources;

use App\Application\HTTP\Response\ResourceCollection;

final class DeliveryCollection extends ResourceCollection
{
    public function __construct(
        iterable $data,
    ) {
        parent::__construct($data, DeliveryResource::class);
    }
}

