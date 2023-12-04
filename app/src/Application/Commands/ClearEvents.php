<?php

declare(strict_types=1);

namespace App\Application\Commands;

use Spiral\Cqrs\CommandInterface;

class ClearEvents implements CommandInterface
{
    public function __construct(
        public readonly ?string $type = null,
        public readonly ?array $uuids = null,
    ) {}
}
