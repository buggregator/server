<?php

declare(strict_types=1);

namespace Modules\Webhooks\Application;

use App\Application\Domain\ValueObjects\Uuid;
use Modules\Webhooks\Domain\Webhook;
use Modules\Webhooks\Domain\WebhookFactoryInterface;

final readonly class WebhookFactory implements WebhookFactoryInterface
{
    public function create(
        string $event,
        string $url,
        bool $verifySsl = false,
        bool $retryOnFailure = true,
    ): Webhook {
        return new Webhook(
            uuid: Uuid::generate(),
            event: $event,
            url: $url,
            verifySsl: $verifySsl,
            retryOnFailure: $retryOnFailure,
        );
    }
}
