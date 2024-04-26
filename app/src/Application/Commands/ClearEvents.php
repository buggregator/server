<?php

declare(strict_types=1);

namespace App\Application\Commands;

use Spiral\Cqrs\CommandInterface;

final readonly class ClearEvents implements CommandInterface
{
    public function __construct(
        public ?string $type = null,
        public ?array $uuids = null,
    ) {}
}
