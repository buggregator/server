<?php

declare(strict_types=1);

namespace Modules\Events\Interfaces\Http\Request;

use Spiral\Filters\Attribute\Input\Query;
use Spiral\Filters\Model\Filter;
use Spiral\Filters\Model\FilterDefinitionInterface;
use Spiral\Filters\Model\HasFilterDefinition;
use Spiral\Validator\FilterDefinition;
use OpenApi\Attributes as OA;

#[OA\Schema(schema: 'EventsRequest')]
final class EventsRequest extends Filter implements HasFilterDefinition
{
    #[OA\Parameter(
        name: 'type',
        description: 'Event type',
        required: false,
        schema: new OA\Schema(type: 'string'),
    )]
    #[Query]
    public ?string $type = null;

    #[OA\Parameter(
        name: 'project',
        description: 'Event project',
        required: false,
        schema: new OA\Schema(type: 'string'),
    )]
    #[Query]
    public ?string $project = null;


    public function filterDefinition(): FilterDefinitionInterface
    {
        return new FilterDefinition([
            'type' => ['string'],
            'project' => ['string'],
        ]);
    }
}
