<?php

declare(strict_types=1);

namespace Modules\Profiler\Integration\CycleOrm;

use App\Application\Domain\ValueObjects\Uuid;
use Modules\Profiler\Domain\Edge;
use Modules\Profiler\Domain\EdgeFactoryInterface;

final readonly class EdgeFactory implements EdgeFactoryInterface
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
