<?php

declare(strict_types=1);

namespace App\Application\Persistence;

use App\Application\Domain\ValueObjects\Uuid;
use Modules\SshTunnel\Domain\Connection;
use Modules\SshTunnel\Domain\ConnectionRepositoryInterface;

final class ArraySshConnectionRepository implements ConnectionRepositoryInterface
{
    public function __construct(
        private array $connections = [],
    ) {}

    public function store(Connection $connection): bool
    {
        $this->connections[(string)$connection->uuid] = $connection;
        return true;
    }

    public function deleteByPK(Uuid $uuid): bool
    {
        if (!isset($this->connections[(string)$uuid])) {
            return false;
        }

        unset($this->connections[(string)$uuid]);
        return true;
    }

    /**
     * @param Uuid|string $id
     */
    public function findByPK(mixed $id): ?object
    {
        if (!isset($this->connections[(string)$id])) {
            return null;
        }

        return $this->connections[(string)$id];
    }

    public function findOne(array $scope = []): ?object
    {
        throw new \BadMethodCallException('Not available');
    }

    public function findAll(array $scope = []): iterable
    {
        return $this->connections;
    }
}
