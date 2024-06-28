<?php

declare(strict_types=1);

namespace Modules\Profiler\Domain;

use Modules\Profiler\Domain\Edge\Cost;
use Modules\Profiler\Domain\Edge\Diff;
use Modules\Profiler\Domain\Edge\Percents;
use App\Application\Domain\ValueObjects\Uuid;

interface EdgeFactoryInterface
{
    public function create(
        Uuid $profileUuid,
        int $order,
        Cost $cost,
        Diff $diff,
        Percents $percents,
        string $callee,
        ?string $caller,
        ?Uuid $parentUuid,
    ): Edge;
}
