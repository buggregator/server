<?php

declare(strict_types=1);

namespace Modules\Projects\Domain;

interface ProjectFactoryInterface
{
    public function create(string $key, string $name): Project;
}
