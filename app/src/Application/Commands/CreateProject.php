<?php

declare(strict_types=1);

namespace App\Application\Commands;

use Spiral\Cqrs\CommandInterface;

final readonly class CreateProject implements CommandInterface
{
    public function __construct(
        public string $key,
        public string $name,
    ) {}
}
