<?php

declare(strict_types=1);

namespace Modules\Profiler\Integration\CycleOrm;

use App\Application\Domain\ValueObjects\Uuid;
use Cycle\ORM\Select\Repository;
use Modules\Profiler\Domain\ProfileEdgeRepositoryInterface;

final class ProfileEdgeRepository extends Repository implements ProfileEdgeRepositoryInterface
{
    public function getByProfileUuid(Uuid $profileUuid): iterable
    {
        return $this->select()
            ->where('profile_uuid', $profileUuid)
            ->orderBy('order', 'ASC')
            ->fetchAll();
    }
}
