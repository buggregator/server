<?php

declare(strict_types=1);

namespace App\Application\Commands;

use Spiral\Cqrs\QueryInterface;

/**
 * @implements QueryInterface<mixed>
 */
final class FindEventsCursor extends AskEvents implements QueryInterface
{
    public function __construct(
        ?string $type = null,
        ?string $project = null,
        public readonly mixed $limit = null,
        public readonly mixed $cursor = null,
    ) {
        parent::__construct($type, $project);
    }
}
