<?php

declare(strict_types=1);

namespace Modules\Events\Interfaces\Http\Request;

use Spiral\Filters\Attribute\Input\Data;
use Spiral\Filters\Model\Filter;
use Spiral\Filters\Model\FilterDefinitionInterface;
use Spiral\Filters\Model\HasFilterDefinition;
use Spiral\Validator\FilterDefinition;
use OpenApi\Attributes as OA;

#[OA\Schema(schema: 'ClearEventsRequest')]
final class ClearEventsRequest extends Filter implements HasFilterDefinition
{
    #[OA\Property(
        property: 'type',
        description: 'Event type',
        type: 'string',
        nullable: true,
    )]
    #[Data]
    public ?string $type = null;

    #[OA\Property(
        property: 'project',
        description: 'Event project',
        type: 'string',
        nullable: true,
    )]
    #[Data]
    public ?string $project = null;

    #[OA\Property(
        property: 'uuids',
        description: 'Uuids',
        type: 'array',
        items: new OA\Items(type: 'string', format: 'uuid'),
        nullable: true,
    )]
    #[Data]
    public ?array $uuids = null;

    public function filterDefinition(): FilterDefinitionInterface
    {
        return new FilterDefinition([
            'type' => ['string'],
            'project' => ['string'],
            'uuids' => ['array'],
        ]);
    }
}
