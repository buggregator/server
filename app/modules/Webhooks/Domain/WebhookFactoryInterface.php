<?php

declare(strict_types=1);

namespace Modules\Webhooks\Domain;

use Ramsey\Uuid\UuidInterface;

interface WebhookFactoryInterface
{
    public function create(
        UuidInterface $uuid,
        string $event,
        string $url,
        bool $verifySsl = false,
        bool $retryOnFailure = true,
    ): Webhook;
}
