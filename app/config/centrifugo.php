<?php

declare(strict_types=1);

use App\Interfaces\Centrifugo\ConnectService;
use App\Interfaces\Centrifugo\RPCService;
use App\Interfaces\Centrifugo\SubscribeService;
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
