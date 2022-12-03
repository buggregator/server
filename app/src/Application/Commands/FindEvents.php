<?php

declare(strict_types=1);

namespace App\Application\Commands;

final class FindEvents extends AskEvents
{
    public function __construct(
        public readonly ?string $type = null,
        public readonly ?int $projectId = null,
        public readonly ?int $offset = null,
        public readonly ?int $limit = null,
    ) {
        parent::__construct($this->type, $this->projectId);
    }
}
