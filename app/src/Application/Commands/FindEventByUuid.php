<?php

declare(strict_types=1);

namespace App\Application\Commands;

use App\Application\Domain\ValueObjects\Uuid;
use Spiral\Cqrs\QueryInterface;

final readonly class FindEventByUuid implements QueryInterface
{
    public function __construct(
        public Uuid $uuid
    ) {}
}
