<?php

declare(strict_types=1);

namespace Modules\Webhooks\Application;

use Modules\Webhooks\Domain\Webhook;
use Modules\Webhooks\Domain\WebhookFactoryInterface;
use Ramsey\Uuid\UuidInterface;

final readonly class WebhookFactory implements WebhookFactoryInterface
{
    public function create(
        UuidInterface $uuid,
        string $event,
        string $url,
        bool $verifySsl = false,
        bool $retryOnFailure = true,
    ): Webhook {
        return new Webhook(
            uuid: $uuid,
            event: $event,
            url: $url,
            verifySsl: $verifySsl,
            retryOnFailure: $retryOnFailure,
        );
    }
}
