<?php

declare(strict_types=1);

namespace App\Integration\MongoDb\Persistence;

trait HelpersTrait
{
    protected function makeOptions(array $orderBy = [], int $limit = 30, int $offset = 0): array
    {
        return [
            'sort' => $this->mapOrderBy($orderBy),
            'limit' => $limit,
            'skip' => $offset,
        ];
    }

    protected function mapOrderBy(array $orderBy): array
    {
        $result = [];

        foreach ($orderBy as $key => $order) {
            $result[$key] = $order === 'asc' ? 1 : -1;
        }

        return $result;
    }
}
