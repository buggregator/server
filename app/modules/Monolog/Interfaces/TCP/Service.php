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

final readonly class Service implements ServiceInterface
{
    public function __construct(
        private CommandBusInterface $commandBus,
        private LoggerInterface $logger,
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
                /** @var array{
                 *     message: string,
                 *     context: array{project?: string},
                 *     level: int,
                 *     level_name: string,
                 *     channel: string,
                 *     datetime: string,
                 *     extra: array,
                 * } $payload
                 */
                $payload = \json_decode($message, true, JSON_THROW_ON_ERROR);

                $project = $payload['context']['project'] ?? null;
                if ($project !== null && !\is_string($project)) {
                    $this->logger->warning("[Monolog] Project must be a string", [
                        'message' => $message,
                    ]);

                    $project = null;
                }

                $this->commandBus->dispatch(
                    new HandleReceivedEvent(
                        type: 'monolog',
                        payload: $payload,
                        project: $project,
                    ),
                );
            } catch (\JsonException) {
                // Impossible to decode the message, give up.
                $this->logger->error("[Monolog] Unable to decode log message. Should be a valid JSON.", [
                    'message' => $message,
                ]);
            }
        }

        return new ContinueRead();
    }
}
