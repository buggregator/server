<?php

declare(strict_types=1);

namespace Modules\Profiler\Interfaces\Http\Request;

use Spiral\Filters\Attribute\Input\Query;
use Spiral\Filters\Model\Filter;
use Spiral\Filters\Model\FilterDefinitionInterface;
use Spiral\Filters\Model\HasFilterDefinition;
use Spiral\Validator\FilterDefinition;

final class CompareProfilesRequest extends Filter implements HasFilterDefinition
{
    #[Query]
    public string $base = '';

    #[Query]
    public string $compare = '';

    public function filterDefinition(): FilterDefinitionInterface
    {
        return new FilterDefinition([
            'base' => [
                ['notEmpty'],
                ['string::length', 36, 36],
            ],
            'compare' => [
                ['notEmpty'],
                ['string::length', 36, 36],
            ],
        ]);
    }
}
