<?php

declare(strict_types=1);

namespace Modules\Webhooks\Domain;

use Ramsey\Uuid\UuidInterface;

interface WebhookServiceInterface
{
    public function send(WebhookEvent $event): void;

    public function sendWebhook(UuidInterface $uuid, WebhookEvent $event): void;
}
