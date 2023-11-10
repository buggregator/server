<?php

declare(strict_types=1);

namespace Modules\Monolog\Interfaces\TCP;

use App\Application\Commands\HandleReceivedEvent;
use Spiral\Cqrs\CommandBusInterface;
use Spiral\RoadRunner\Tcp\Request;
use Spiral\RoadRunner\Tcp\TcpEvent;
use Spiral\RoadRunnerBridge\Tcp\Response\CloseConnection;
use Spiral\RoadRunnerBridge\Tcp\Response\ContinueRead;
use Spiral\RoadRunnerBridge\Tcp\Response\ResponseInterface;
use Spiral\RoadRunnerBridge\Tcp\Service\ServiceInterface;

class Service implements ServiceInterface
{
    public function __construct(
        private readonly CommandBusInterface $commandBus,
    ) {
    }

    public function handle(Request $request): ResponseInterface
    {
        if ($request->event === TcpEvent::Connected) {
            return new ContinueRead();
        }

        if ($request->event === TcpEvent::Close) {
            return new CloseConnection();
        }

        $messages = \array_filter(\explode("\n", $request->body));

        foreach ($messages as $message) {
            $payload = \json_decode($message, true);

            // Impossible to decode the message, give up.
            if (!$payload) {
                throw new \RuntimeException("Unable to decode a message from [{$request->connectionUuid}] client.");
            }

            $this->commandBus->dispatch(
                new HandleReceivedEvent(
                    type: 'monolog',
                    payload: $payload
                )
            );
        }

        return new ContinueRead();
    }
}
