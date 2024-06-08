<?php

declare(strict_types=1);

namespace Modules\Profiler\Domain;

use Modules\Profiler\Domain\Profile\Peaks;

interface ProfileFactoryInterface
{
    public function create(
        string $name,
        array $tags,
        Peaks $peaks,
    ): Profile;
}
