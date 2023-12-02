<?php

declare(strict_types=1);

namespace App\Interfaces\Centrifugo;

use App\Application\Broadcasting\Channel\EventsChannel;
use App\Application\Broadcasting\Channel\SettingsChannel;
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
                    user: (string)$request->getAttribute('user_id'),
                    channels: [
                        (string)new EventsChannel(),
                        (string)new SettingsChannel(),
                    ],
                ),
            );
        } catch (\Throwable $e) {
            $request->error($e->getCode(), $e->getMessage());
        }
    }
}
