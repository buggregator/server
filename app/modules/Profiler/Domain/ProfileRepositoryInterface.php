<?php

declare(strict_types=1);

namespace Modules\Profiler\Domain;

use App\Application\Domain\ValueObjects\Uuid;
use App\Application\Exception\EntityNotFoundException;
use Cycle\ORM\RepositoryInterface;

/**
 * @template TEntity of Profile
 * @extends RepositoryInterface<TEntity>
 */
interface ProfileRepositoryInterface extends RepositoryInterface
{
    /**
     * @throws EntityNotFoundException
     */
    public function getByUuid(Uuid $uuid): Profile;
}
