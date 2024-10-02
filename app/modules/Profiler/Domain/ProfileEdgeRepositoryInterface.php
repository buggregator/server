<?php

declare(strict_types=1);

namespace Modules\Profiler\Domain;

use App\Application\Domain\ValueObjects\Uuid;
use Cycle\ORM\RepositoryInterface;

/**
 * @template TEntity of Edge
 * @extends RepositoryInterface<TEntity>
 */
interface ProfileEdgeRepositoryInterface extends RepositoryInterface
{
    /**
     * @return iterable<TEntity>
     */
    public function getByProfileUuid(Uuid $profileUuid): iterable;
}
