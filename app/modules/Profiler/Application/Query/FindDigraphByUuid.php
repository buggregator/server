<?php

declare(strict_types=1);

namespace Modules\Profiler\Application\Query;

use App\Application\Domain\ValueObjects\Uuid;
use Spiral\Cqrs\QueryInterface;

final readonly class FindDigraphByUuid implements QueryInterface
{
    public function __construct(
        public Uuid $profileUuid,
        public float $threshold = 0.01,
        public bool $criticalPath = true,
    ) {}
}
