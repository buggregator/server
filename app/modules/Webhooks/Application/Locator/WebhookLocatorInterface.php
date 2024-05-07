<?php

declare(strict_types=1);

namespace Modules\Webhooks\Application\Locator;

interface WebhookLocatorInterface
{
    /**
     * @return iterable<Webhook>
     */
    public function findAll(): iterable;
}
