<?php

declare(strict_types=1);

namespace Modules\Profiler\Interfaces\Http\Request;

use Modules\Profiler\Application\TopFunctions\Metric;
use OpenApi\Attributes as OA;
use Spiral\Filters\Attribute\Input\Query;
use Spiral\Filters\Model\Filter;
use Spiral\Filters\Model\FilterDefinitionInterface;
use Spiral\Filters\Model\HasFilterDefinition;
use Spiral\Validator\FilterDefinition;

#[OA\Schema(schema: 'TopFunctionsRequest')]
final class TopFunctionsRequest extends Filter implements HasFilterDefinition
{
    #[OA\Property(
        property: 'limit',
        description: 'Limit the number of results',
        type: 'integer',
    )]
    #[Query]
    public int $limit = 100;

    #[OA\Property(
        property: 'metric',
        description: 'Metric',
        type: 'string',
    )]
    #[Query]
    public Metric $metric = Metric::CPU;

    public function filterDefinition(): FilterDefinitionInterface
    {
        return new FilterDefinition([
            'limit' => [
                ['number::range', 10, 300],
            ],
        ]);
    }
}
