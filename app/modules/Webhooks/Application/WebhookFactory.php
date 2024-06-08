<?php

declare(strict_types=1);

namespace Modules\Webhooks\Application;

use App\Application\Domain\ValueObjects\Json;
use App\Application\Domain\ValueObjects\Uuid;
use Modules\Webhooks\Domain\ValueObject\Url;
use Modules\Webhooks\Domain\Webhook;
use Modules\Webhooks\Domain\WebhookFactoryInterface;

final readonly class WebhookFactory implements WebhookFactoryInterface
{
    public function create(
        string $key,
        string $event,
        Url $url,
        Json $headers = new Json(),
        bool $verifySsl = false,
        bool $retryOnFailure = true,
    ): Webhook {
        return new Webhook(
            uuid: Uuid::generate(),
            key: $key,
            event: $event,
            url: $url,
            headers: $headers,
            verifySsl: $verifySsl,
            retryOnFailure: $retryOnFailure,
        );
    }
}
