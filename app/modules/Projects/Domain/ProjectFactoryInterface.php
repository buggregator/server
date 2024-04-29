<?php

declare(strict_types=1);

namespace Modules\Projects\Domain;

use Modules\Projects\Domain\ValueObject\Key;

interface ProjectFactoryInterface
{
    public function create(Key $key, string $name): Project;
}
