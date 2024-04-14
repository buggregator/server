<?php

declare(strict_types=1);

namespace App\Application\Commands;

final class FindEvents extends AskEvents
{
    public function __construct(
        ?string $type = null,
        ?int $projectId = null,
        public readonly int $limit = 100,
    ) {
        parent::__construct($type, $projectId);
    }
}
