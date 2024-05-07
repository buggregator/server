<?php

declare(strict_types=1);

namespace Modules\Webhooks\Application\Locator;

use App\Application\Domain\ValueObjects\Uuid;
use Modules\Webhooks\Exceptions\WebhooksAlreadyExistsException;

interface WebhookRegistryInterface
{
    /**
     * @throws WebhooksAlreadyExistsException
     */
    public function register(Webhook $webhook): Uuid;
}
