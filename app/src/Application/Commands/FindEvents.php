<?php

declare(strict_types=1);

namespace App\Application\Commands;

use Modules\Events\Domain\Event;
use Spiral\Cqrs\QueryInterface;

/**
 * @implements QueryInterface<Event[]>
 */
final class FindEvents extends AskEvents
{
    public function __construct(
        ?string $type = null,
        ?string $project = null,
        public readonly int $limit = 100,
    ) {
        parent::__construct($type, $project);
    }
}
