<?php

declare(strict_types=1);

namespace App\Application\Commands;

use Spiral\Cqrs\QueryInterface;

class FindProjectByName implements QueryInterface
{
    public function __construct(
        public readonly string $name
    ) {
    }
}
