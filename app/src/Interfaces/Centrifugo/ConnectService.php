<?php

declare(strict_types=1);

namespace App\Api\Centrifugo;

use RoadRunner\Centrifugo\Payload\ConnectResponse;
use RoadRunner\Centrifugo\Request;
use RoadRunner\Centrifugo\Request\RequestInterface;
use Spiral\RoadRunnerBridge\Centrifugo\ServiceInterface;

class ConnectService implements ServiceInterface
{
    /**
     * @param Request\Connect $request
     */
    public function handle(RequestInterface $request): void
    {
        try {
            $request->respond(
                new ConnectResponse(
                    user: (string) $request->getAttribute('user_id')
                )
            );
        } catch (\Throwable $e) {
            $request->error($e->getCode(), $e->getMessage());
        }
    }
}
