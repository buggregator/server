<?php

declare(strict_types=1);

use App\Api\Centrifugo\ConnectService;
use App\Api\Centrifugo\RPCService;
use App\Api\Centrifugo\SubscribeService;
use RoadRunner\Centrifugo\Request\RequestType;

return [
    'services' => [
        RequestType::Connect->value => ConnectService::class,
        RequestType::Subscribe->value => SubscribeService::class,
        RequestType::RPC->value => RPCService::class,
    ],
    'interceptors' => [
        '*' => [],
    ],
];
