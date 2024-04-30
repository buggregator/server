<?php

declare(strict_types=1);

namespace Modules\Metrics\Application;

use Spiral\RoadRunner\Metrics\CollectorInterface;

interface CollectorRepositoryInterface
{
    /**
     * @return iterable<non-empty-string, CollectorInterface>
     */
    public function findAll(): iterable;
}
