<?php

declare(strict_types=1);

namespace Modules\Profiler\Integration\CycleOrm;

use App\Application\Domain\ValueObjects\Uuid;
use App\Application\Exception\EntityNotFoundException;
use Cycle\ORM\Select\Repository;
use Modules\Profiler\Domain\Profile;
use Modules\Profiler\Domain\ProfileRepositoryInterface;

/**
 * @extends ProfileRepositoryInterface<Profile>
 */
final class ProfileRepository extends Repository implements ProfileRepositoryInterface
{
    public function getByUuid(Uuid $uuid): Profile
    {
        $profile = $this->findByPK($uuid);

        if ($profile === null) {
            throw new EntityNotFoundException('Profile not found');
        }

        return $profile;
    }
}
