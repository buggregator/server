<?php

declare(strict_types=1);

namespace App\Interfaces\Centrifugo;

use RoadRunner\Centrifugo\Payload\SubscribeResponse;
use RoadRunner\Centrifugo\Request;
use RoadRunner\Centrifugo\Request\RequestInterface;
use Spiral\RoadRunnerBridge\Centrifugo\ServiceInterface;

final class SubscribeService implements ServiceInterface
{
    public function handle(RequestInterface $request): void
    {
        \assert($request instanceof Request\Subscribe);

        try {
            $request->respond(
                new SubscribeResponse(),
            );
        } catch (\Throwable $e) {
            $request->error((int) $e->getCode(), $e->getMessage());
        }
    }
}
