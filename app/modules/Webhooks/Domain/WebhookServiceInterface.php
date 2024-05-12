<?php

declare(strict_types=1);

namespace Modules\Webhooks\Domain;

interface WebhookServiceInterface
{
    public function send(WebhookEvent $event): void;
}
