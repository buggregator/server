<?php

declare(strict_types=1);

namespace Modules\Profiler\Domain\Edge;

use Cycle\Annotated\Annotation\Column;
use Cycle\Annotated\Annotation\Embeddable;

#[Embeddable(
    columnPrefix: 'd_',
)]
class Diff
{
    public function __construct(
        #[Column(type: 'integer')]
        public int $cpu,
        #[Column(type: 'integer')]
        public int $wt,
        #[Column(type: 'integer')]
        public int $ct,
        #[Column(type: 'integer')]
        public int $mu,
        #[Column(type: 'integer')]
        public int $pmu,
    ) {}
}
