<?php

declare(strict_types=1);

namespace Modules\Webhooks\Domain;

use App\Application\Domain\Entity\Json;
use Modules\Webhooks\Domain\ValueObject\Url;

interface WebhookFactoryInterface
{
    public function create(
        string $key,
        string $event,
        Url $url,
        Json $headers = new Json(),
        bool $verifySsl = false,
        bool $retryOnFailure = true,
    ): Webhook;
}
