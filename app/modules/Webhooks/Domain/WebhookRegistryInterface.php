<?php

declare(strict_types=1);

namespace Modules\Webhooks\Domain;

interface WebhookRegistryInterface
{
    public function register(Webhook $webhook): void;
}
