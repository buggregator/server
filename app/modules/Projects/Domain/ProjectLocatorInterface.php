<?php

declare(strict_types=1);

namespace Modules\Projects\Domain;

interface ProjectLocatorInterface
{
    /**
     * @return iterable<Project>
     */
    public function findAll(): iterable;
}
