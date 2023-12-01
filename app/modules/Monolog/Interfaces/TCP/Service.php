<?php

declare(strict_types=1);

namespace Modules\Monolog\Interfaces\TCP;

use App\Application\Commands\HandleReceivedEvent;
use Psr\Log\LoggerInterface;
use Spiral\Cqrs\CommandBusInterface;
use Spiral\RoadRunner\Tcp\Request;
use Spiral\RoadRunner\Tcp\TcpEvent;
use Spiral\RoadRunnerBridge\Tcp\Response\CloseConnection;
use Spiral\RoadRunnerBridge\Tcp\Response\ContinueRead;
use Spiral\RoadRunnerBridge\Tcp\Response\ResponseInterface;
use Spiral\RoadRunnerBridge\Tcp\Service\ServiceInterface;

final class Service implements ServiceInterface
{
    public function __construct(
        private readonly CommandBusInterface $commandBus,
        private readonly LoggerInterface $logger,
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
            try {
                $payload = \json_decode($message, true, JSON_THROW_ON_ERROR);
                $this->commandBus->dispatch(
                    new HandleReceivedEvent(
                        type: 'monolog',
                        payload: $payload,
                    ),
                );
            } catch (\JsonException $e) {
                // Impossible to decode the message, give up.
                $this->logger->error("Unable to decode log message. Should be a valid JSON.", [
                    'message' => $message,
                ]);
            }
        }

        return new ContinueRead();
    }
}
