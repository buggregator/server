<?php

declare(strict_types=1);

namespace Modules\Webhooks\Domain;

final readonly class WebhookEvent
{
    public function __construct(
        public string $event,
        public array $payload,
    ) {}
}
