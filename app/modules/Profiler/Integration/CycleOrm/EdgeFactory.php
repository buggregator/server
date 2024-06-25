<?php

declare(strict_types=1);

namespace Modules\Profiler\Integration\CycleOrm;

use Modules\Profiler\Domain\Edge\Cost;
use Modules\Profiler\Domain\Edge\Diff;
use Modules\Profiler\Domain\Edge\Percents;
use App\Application\Domain\ValueObjects\Uuid;
use Modules\Profiler\Domain\Edge;
use Modules\Profiler\Domain\EdgeFactoryInterface;

final readonly class EdgeFactory implements EdgeFactoryInterface
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
    ): Edge {
        return new Edge(
            uuid: Uuid::generate(),
            profileUuid: $profileUuid,
            order: $order,
            cost: $cost,
            diff: $diff,
            percents: $percents,
            callee: $callee,
            caller: $caller,
            parentUuid: $parentUuid,
        );
    }
}
