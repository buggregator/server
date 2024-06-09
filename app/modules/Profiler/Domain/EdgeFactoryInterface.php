<?php

declare(strict_types=1);

namespace Modules\Profiler\Domain;

use App\Application\Domain\ValueObjects\Uuid;

interface EdgeFactoryInterface
{
    public function create(
        Uuid $profileUuid,
        int $order,
        Edge\Cost $cost,
        Edge\Diff $diff,
        Edge\Percents $percents,
        string $callee,
        ?string $caller,
        ?Uuid $parentUuid,
    ): Edge;
}
