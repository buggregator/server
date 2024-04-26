<?php

declare(strict_types=1);

namespace Modules\Webhooks\Domain;

interface WebhookFactoryInterface
{
    public function create(
        string $event,
        string $url,
        bool $verifySsl = false,
        bool $retryOnFailure = true,
    ): Webhook;
}
