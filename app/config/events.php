<?php

declare(strict_types=1);

use Modules\Webhooks\Application\Broadcasting\WebhookEventInterceptor;
use App\Application\Broadcasting\BroadcastEventInterceptor;

return [
    'interceptors' => [
        WebhookEventInterceptor::class,
        BroadcastEventInterceptor::class,
    ],
];
