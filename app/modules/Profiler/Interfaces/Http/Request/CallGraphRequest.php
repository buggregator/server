<?php

declare(strict_types=1);

namespace Modules\Profiler\Interfaces\Http\Request;

use Modules\Profiler\Application\CallGraph\Metric;
use OpenApi\Attributes as OA;
use Spiral\Filters\Attribute\Input\Query;
use Spiral\Filters\Model\Filter;
use Spiral\Filters\Model\FilterDefinitionInterface;
use Spiral\Filters\Model\HasFilterDefinition;
use Spiral\Validator\FilterDefinition;

#[OA\Schema(schema: 'CallGraphRequest')]
final class CallGraphRequest extends Filter implements HasFilterDefinition
{
    #[OA\Property(
        property: 'threshold',
        description: 'Threshold',
        type: 'integer',
    )]
    #[Query]
    public int|float $threshold = 0;

    #[OA\Property(
        property: 'percentage',
        description: 'Percentage',
        type: 'integer',
    )]
    #[Query]
    public int $percentage = 10;

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
            'threshold' => [
                ['number::range', 0, 100],
            ],
            'percentage' => [
                ['number::range', 0, 100],
            ],
        ]);
    }
}
