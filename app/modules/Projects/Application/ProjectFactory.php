<?php

declare(strict_types=1);

namespace Modules\Projects\Application;

use Modules\Projects\Domain\Project;
use Modules\Projects\Domain\ProjectFactoryInterface;
use Modules\Projects\Domain\ValueObject\Key;

final readonly class ProjectFactory implements ProjectFactoryInterface
{
    public function create(Key $key, string $name): Project
    {
        return new Project($key, $name);
    }
}
