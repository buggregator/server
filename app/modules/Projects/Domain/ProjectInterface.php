<?php

declare(strict_types=1);

namespace Modules\Projects\Domain;

use Modules\Projects\Domain\ValueObject\Key;

interface ProjectInterface
{
    public function getKey(): Key;
    public function getName(): string;
}
