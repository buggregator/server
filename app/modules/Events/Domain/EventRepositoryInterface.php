<?php

declare(strict_types=1);

namespace Modules\Events\Domain;

use Cycle\ORM\RepositoryInterface;

/**
 * @template TEntity of Event
 * @extends RepositoryInterface<Event>
 */
interface EventRepositoryInterface extends RepositoryInterface
{
    public function findAll(array $scope = [], array $orderBy = [], int $limit = 30, int $offset = 0): iterable;

    public function countAll(array $scope = []): int;

    public function store(Event $event): bool;

    public function deleteAll(array $scope = []): void;

    public function deleteByPK(string $uuid): bool;
}
