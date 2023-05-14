<?php

declare(strict_types=1);

namespace App\Interfaces\TCP;

use App\Application\Service\ClientProxy\EventHandlerRegistryInterface;
use Psr\Log\LoggerInterface;
use Spiral\RoadRunner\Tcp\Request;
use Spiral\RoadRunner\Tcp\TcpWorkerInterface;
use Spiral\RoadRunnerBridge\Tcp\Response\ContinueRead;
use Spiral\RoadRunnerBridge\Tcp\Response\ResponseInterface;
use Spiral\RoadRunnerBridge\Tcp\Service\ServiceInterface;

final class ClientProxyService implements ServiceInterface
{
    public function __construct(
        private readonly EventHandlerRegistryInterface $handlerRegistry,
        private readonly LoggerInterface $logger,
    ) {
    }

    public function handle(Request $request): ResponseInterface
    {
        if ($request->event === TcpWorkerInterface::EVENT_CONNECTED) {
            return new ContinueRead();
        }

        $messages = \json_decode($request->body, true, 512, JSON_THROW_ON_ERROR);

        foreach ($messages as $message) {
            try {
                $this->handlePayload($message);
            } catch (\Throwable $e) {
                $this->logger->debug($e->getMessage());
            }
        }

        return new ContinueRead();
    }

    /**
     * @param array{type: string, data: string, time: string} $payload
     */
    private function handlePayload(array $payload): void
    {
        $this->handlerRegistry->handle($payload['type'], $payload['data']);
    }
}
