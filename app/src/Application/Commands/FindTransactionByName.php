<?php

declare(strict_types=1);

namespace App\Application\Commands;

use Spiral\Cqrs\QueryInterface;

class FindTransactionByName implements QueryInterface
{
    public function __construct(
        public readonly string $name
    ) {}
}
