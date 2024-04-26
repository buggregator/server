<?php

declare(strict_types=1);

namespace Modules\Webhooks\Domain;

interface WebhookLocatorInterface
{
    /**
     * @return iterable<Webhook>
     */
    public function findAll(): iterable;
}
