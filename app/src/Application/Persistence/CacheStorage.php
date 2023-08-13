<?php

declare(strict_types=1);

namespace App\Application\Persistence;

use App\Application\Domain\ValueObjects\Uuid;
use Carbon\Carbon;
use Doctrine\Common\Collections\ArrayCollection;
use Doctrine\Common\Collections\Criteria;
use Psr\SimpleCache\CacheInterface;
use Psr\SimpleCache\InvalidArgumentException;

final class CacheStorage
{
    private const EVENT_IDS_KEY = 'ids';

    public function __construct(
        private readonly CacheInterface $cache,
        private readonly string $prefix,
        private readonly int $ttl = 60 * 60 * 2,
    ) {
    }

    /**
     * @throws InvalidArgumentException
     */
    public function findAll(array $scope = [], array $orderBy = [], int $limit = 30, int $offset = 0): iterable
    {
        $items = $this->getFiltered($scope, $orderBy, $limit, $offset);

        foreach ($items as $item) {
            yield $item;
        }
    }

    /**
     * @throws InvalidArgumentException
     */
    public function countAll(array $scope = []): int
    {
        $items = $this->getFiltered($scope);

        return \count($items);
    }


    /**
     * @throws InvalidArgumentException
     */
    public function store(Uuid $uuid, array $index, array $data): bool
    {
        $id = $this->prefix((string)$uuid);
        $ids = $this->getIds();
        $ids[$id] = [
            ...$index,
            'date' => \microtime(true),
        ];

        $this->cache->set($this->prefix(self::EVENT_IDS_KEY), $ids);

        return $this->cache->set(
            $id,
            $data,
            Carbon::now()->addSeconds($this->ttl)->diffAsCarbonInterval()
        );
    }

    /**
     * @throws InvalidArgumentException
     */
    public function findByPK(string $uuid): ?array
    {
        $item = $this->cache->get($this->prefix((string)$uuid));

        if (\is_array($item)) {
            return $item;
        }

        return null;
    }

    /**
     * @throws InvalidArgumentException
     */
    public function findOne(array $scope = []): ?array
    {
        $items = $this->findAll(scope: $scope, limit: 1);

        foreach ($items as $item) {
            return $item;
        }

        return null;
    }

    /**
     * @throws InvalidArgumentException
     */
    public function deleteByPK(string $uuid): bool
    {
        $uuid = $this->prefix($uuid);

        $ids = $this->getIds();
        if (isset($ids[$uuid])) {
            unset($ids[$uuid]);
            $this->cache->set($this->prefix(self::EVENT_IDS_KEY), $ids);
            $this->cache->delete($uuid);

            return true;
        }

        return false;
    }

    /**
     * @throws InvalidArgumentException
     */
    public function deleteAll(array $scope = []): void
    {
        if ($scope !== []) {
            $ids = $this->getFilteredIds($scope);
            $itemIds = $this->getIds();
            foreach ($ids as $id) {
                unset($itemIds[$id]);
            }

            $this->cache->set($this->prefix(self::EVENT_IDS_KEY), $itemIds);
        } else {
            $ids = \array_keys($this->getIds());
            $this->cache->delete($this->prefix(self::EVENT_IDS_KEY));
        }

        $this->cache->deleteMultiple($ids);
    }

    /**
     * @throws InvalidArgumentException
     */
    private function getFiltered(array $scope = [], array $orderBy = [], int $limit = 30, int $offset = 0): array
    {
        $ids = $this->getFilteredIds($scope, $orderBy);
        $ids = \array_slice($ids, $offset, $limit);

        $items = \array_filter($this->cache->getMultiple($ids));
        $result = [];

        foreach ($items as $item) {
            $result[$item['id']] = $item;
        }

        unset($items, $criteria, $ids);

        return $result;
    }

    /**
     * @throws InvalidArgumentException
     */
    private function getFilteredIds(array $scope = [], array $orderBy = []): array
    {
        $criteria = new Criteria();
        foreach ($scope as $key => $value) {
            $criteria->andWhere(Criteria::expr()->eq($key, $value));
        }

        $criteria->orderBy($orderBy);

        $ids = (new ArrayCollection($this->getIds()))->matching($criteria)->toArray();

        return \array_keys($ids);
    }

    /**
     * @return array<non-empty-string, bool>
     * @throws InvalidArgumentException
     */
    private function getIds(): array
    {
        return (array)$this->cache->get(
            $this->prefix(self::EVENT_IDS_KEY),
            []
        );
    }

    private function prefix(string $key): string
    {
        return $this->prefix . ':' . $key;
    }
}
