<?php

declare(strict_types=1);

namespace Modules\Events\Interfaces\Queries;

use App\Application\Commands\FindEventsCursor;
use Cycle\Database\DatabaseInterface;
use Cycle\Database\Driver\MySQL\MySQLDriver;
use Cycle\Database\Driver\Postgres\PostgresDriver;
use Cycle\Database\Driver\SQLite\SQLiteDriver;
use Cycle\Database\Driver\SQLServer\SQLServerDriver;
use Cycle\Database\Injection\Fragment;
use Cycle\ORM\Select;
use Modules\Events\Domain\Event;
use Modules\Events\Domain\EventRepositoryInterface;
use Modules\Events\Interfaces\Http\Pagination\EventsCursor;
use Spiral\Cqrs\Attribute\QueryHandler;
use Spiral\Cqrs\QueryBusInterface;
use Spiral\Filters\Exception\ValidationException;
use Psr\Log\LoggerInterface;

final class FindEventsCursorHandler extends EventsHandler
{
    private const DEFAULT_LIMIT = 100;
    private const MAX_LIMIT = 100;

    public function __construct(
        private readonly EventRepositoryInterface $events,
        private LoggerInterface $logger,
        private readonly DatabaseInterface $db,
        QueryBusInterface $bus,
    ) {
        parent::__construct($bus);
    }

    #[QueryHandler]
    public function __invoke(FindEventsCursor $query): EventsCursorResult
    {
        $limit = $this->resolveLimit($query->limit);
        $cursor = $this->resolveCursor($query->cursor);

        $source = $this->events->select();
        $scope = $this->getScopeFromFindEvents($query);
        if ($scope !== []) {
            $source = $this->applyScope($source, $scope);
        }

        if ($cursor !== null) {
            $source = $this->applyCursor($source, $cursor);
        }

        $items = $this->applyOrder($source)
            ->limit($limit + 1)
            ->fetchAll();

        $hasMore = \count($items) > $limit;
        if ($hasMore) {
            \array_pop($items);
        }

        $nextCursor = null;
        if ($hasMore && $items !== []) {
            $last = $items[\array_key_last($items)];
            if ($last instanceof Event) {
                $nextCursor = EventsCursor::fromEvent($last)->toOpaque();
            }
        }

        return new EventsCursorResult(
            $items,
            $limit,
            $hasMore,
            $nextCursor,
        );
    }

    private function applyScope(Select $select, array $scope): Select
    {
        $normalized = [];

        foreach ($scope as $key => $value) {
            $normalized[$key] = \is_array($value) ? ['in' => $value] : $value;
        }

        return $select->where($normalized);
    }

    private function applyCursor(Select $select, EventsCursor $cursor): Select
    {
        $timestampExpression = $this->timestampExpression();

        return $select->where(
            static function ($query) use ($cursor, $timestampExpression): void {
                $query
                    ->where(new Fragment($timestampExpression), '<', $cursor->getTimestamp())
                    ->orWhere(
                        static function ($query) use ($cursor, $timestampExpression): void {
                            $query
                                ->where(new Fragment($timestampExpression), '=', $cursor->getTimestamp())
                                ->andWhere(Event::UUID, '<', $cursor->getUuid());
                        },
                    );
            },
        );
    }

    private function applyOrder(Select $select): Select
    {
        $select = $select->orderBy(new Fragment($this->timestampExpression()), 'DESC');

        return $select->orderBy(Event::UUID, 'DESC');
    }

    private function timestampExpression(): string
    {
        $driver = $this->db->getDriver();
        $table = $driver->identifier(Event::ROLE);
        $column = $driver->identifier(Event::TIMESTAMP);
        $expression = "{$table}.{$column}";

        return match (true) {
            $driver instanceof MySQLDriver, $driver instanceof SQLServerDriver =>
                sprintf('CAST(%s AS DECIMAL(20,6))', $expression),

            $driver instanceof SQLiteDriver =>
                sprintf('CAST(%s AS REAL)', $expression),

            $driver instanceof PostgresDriver =>
                sprintf('CAST(%s AS numeric)', $expression),

            default =>
                throw new \RuntimeException('Unsupported DB driver'),
        };
    }

    private function resolveLimit(mixed $value): int
    {
        if ($value === null || $value === '') {
            return self::DEFAULT_LIMIT;
        }

        if (!\is_numeric($value)) {
            throw $this->invalidLimit();
        }

        $limit = (int) $value;
        if ($limit < 1) {
            throw $this->invalidLimit();
        }

        return \min($limit, self::MAX_LIMIT);
    }

    private function resolveCursor(mixed $value): ?EventsCursor
    {
        if ($value === null || $value === '') {
            return null;
        }

        if (!\is_string($value)) {
            throw $this->invalidCursor();
        }

        return EventsCursor::fromOpaque($value);
    }

    private function invalidLimit(): ValidationException
    {
        return new ValidationException([
            'limit' => ['Limit must be a positive integer.'],
        ], 'Invalid pagination limit.');
    }

    private function invalidCursor(): ValidationException
    {
        return new ValidationException([
            'cursor' => ['Cursor must be a string.'],
        ], 'Invalid cursor.');
    }
}
