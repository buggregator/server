<?php

declare(strict_types=1);

namespace Modules\Webhooks\Application\Locator;

interface WebhookFilesFinderInterface
{
    /**
     * @return iterable<Webhook>
     */
    public function find(): iterable;
}
