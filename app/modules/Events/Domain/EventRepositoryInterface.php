<?php

declare(strict_types=1);

namespace Modules\Events\Domain;

use Cycle\ORM\RepositoryInterface;

/**
 * @template TEntity of Event
 */
interface EventRepositoryInterface extends RepositoryInterface
{
    public function findAll(array $scope = [], array $orderBy = []): iterable;

    public function countAll(array $scope = []): int;

    public function store(Event $event): bool;

    public function deleteAll(array $scope = []): void;

    public function deleteByPK(string $uuid): bool;
}
