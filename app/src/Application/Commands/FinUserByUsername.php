<?php

declare(strict_types=1);

namespace App\Application\Commands;

use Spiral\Cqrs\QueryInterface;

final readonly class FinUserByUsername implements QueryInterface
{
    public function __construct(
        public string $username,
    ) {}
}
