<?php

declare(strict_types=1);

namespace Modules\SshTunnel\Domain;

use App\Application\Domain\ValueObjects\Uuid;
use Cycle\ORM\RepositoryInterface;

/**
 * @extends RepositoryInterface<Connection>
 */
interface ConnectionRepositoryInterface extends RepositoryInterface
{
    /**
     * Create connection entity.
     */
    public function store(Connection $connection): bool;

    /**
     * Delete connection entity by primary key.
     */
    public function deleteByPK(Uuid $uuid): bool;
}
