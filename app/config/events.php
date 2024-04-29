<?php

declare(strict_types=1);

return [
    'interceptors' => [
        \Modules\Webhooks\Application\Broadcasting\WebhookEventInterceptor::class,
        \App\Application\Broadcasting\BroadcastEventInterceptor::class,
    ]
];
