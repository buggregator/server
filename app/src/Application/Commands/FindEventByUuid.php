<?php

declare(strict_types=1);

namespace App\Application\Commands;

use App\Application\Exception\EntityNotFoundException;
use App\Application\Domain\ValueObjects\Uuid;
use Modules\Events\Domain\Event;
use Spiral\Cqrs\QueryInterface;

/**
 * @implements QueryInterface<Event>
 *
 * @throws EntityNotFoundException if event with given uuid was not found
 */
final readonly class FindEventByUuid implements QueryInterface
{
    public function __construct(
        public Uuid $uuid,
    ) {}
}
