<?php

declare(strict_types=1);

namespace Modules\Profiler\Application\Query;

use App\Application\Domain\ValueObjects\Uuid;
use Spiral\Cqrs\QueryInterface;

final class CompareProfiles implements QueryInterface
{
    public function __construct(
        public Uuid $baseProfileUuid,
        public Uuid $compareProfileUuid,
        public int $limit = 50,
    ) {}
}
