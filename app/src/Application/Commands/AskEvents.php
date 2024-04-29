<?php

declare(strict_types=1);

namespace App\Application\Commands;

use Spiral\Cqrs\QueryInterface;

abstract class AskEvents implements QueryInterface
{
    public function __construct(
        public readonly ?string $type = null,
        public readonly ?string $project = null,
    ) {}
}
