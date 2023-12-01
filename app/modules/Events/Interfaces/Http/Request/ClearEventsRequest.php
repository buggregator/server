<?php

declare(strict_types=1);

namespace Modules\Events\Interfaces\Http\Request;

use Spiral\Filters\Attribute\Input\Data;
use Spiral\Filters\Model\Filter;
use Spiral\Filters\Model\FilterDefinitionInterface;
use Spiral\Filters\Model\HasFilterDefinition;
use Spiral\Validator\FilterDefinition;

final class ClearEventsRequest extends Filter implements HasFilterDefinition
{
    #[Data]
    public ?string $type = null;

    public function filterDefinition(): FilterDefinitionInterface
    {
        return new FilterDefinition([
            'type' => ['string'],
        ]);
    }
}
