<?php

declare(strict_types=1);

namespace Modules\Webhooks\Interfaces\Job;

use Ramsey\Uuid\UuidInterface;

final readonly class JobPayload
{
    public function __construct(
        public UuidInterface $webhookUuid,
        public string $event,
        public array $payload,
    ) {
    }
}
