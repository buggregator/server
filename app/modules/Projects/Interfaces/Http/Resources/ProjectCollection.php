<?php

declare(strict_types=1);

namespace Modules\Projects\Interfaces\Http\Resources;

use App\Application\HTTP\Response\ResourceCollection;

final class ProjectCollection extends ResourceCollection
{
    public function __construct(
        iterable $data,
    ) {
        parent::__construct($data, ProjectResource::class);
    }
}
