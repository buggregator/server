<?php

declare(strict_types=1);

namespace Modules\Profiler\Integration\CycleOrm;

use App\Application\Domain\ValueObjects\Uuid;
use Modules\Profiler\Domain\Profile;
use Modules\Profiler\Domain\Profile\Peaks;
use Modules\Profiler\Domain\ProfileFactoryInterface;

final readonly class ProfileFactory implements ProfileFactoryInterface
{

    public function create(string $name, array $tags, Peaks $peaks): Profile
    {
        return new Profile(
            uuid: Uuid::generate(),
            name: $name,
            peaks: $peaks,
        );
    }
}
