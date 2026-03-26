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

#[OA\Schema(schema: 'FlameChartRequest')]
final class FlameChartRequest extends Filter implements HasFilterDefinition
{
    #[OA\Property(
        property: 'metric',
        description: 'Metric',
        type: 'string',
    )]
    #[Query]
    public Metric $metric = Metric::WallTime;

    public function filterDefinition(): FilterDefinitionInterface
    {
        return new FilterDefinition([]);
    }
}
