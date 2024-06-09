<?php

declare(strict_types=1);

namespace Modules\Profiler\Domain\Edge;

use Cycle\Annotated\Annotation\Column;
use Cycle\Annotated\Annotation\Embeddable;

#[Embeddable(
    columnPrefix: 'p_',
)]
class Percents
{
    public function __construct(
        #[Column(type: 'float')]
        public float $cpu,
        #[Column(type: 'float')]
        public float $wt,
        #[Column(type: 'float')]
        public float $ct,
        #[Column(type: 'float')]
        public float $mu,
        #[Column(type: 'float')]
        public float $pmu,
    ) {}
}
