<?php

declare(strict_types=1);

namespace Modules\Webhooks\Domain;

use App\Application\Domain\ValueObjects\Uuid;

interface WebhookServiceInterface
{
    public function send(WebhookEvent $event): void;

    public function sendWebhook(Uuid $uuid, WebhookEvent $event): void;
}
