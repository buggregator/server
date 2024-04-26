<?php

declare(strict_types=1);

namespace App\Application\Commands;

final class FindEvents extends AskEvents
{
    public function __construct(
        ?string $type = null,
        ?string $project = null,
        public readonly int $limit = 100,
    ) {
        parent::__construct($type, $project);
    }
}
