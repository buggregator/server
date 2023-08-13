<?php

declare(strict_types=1);

namespace App\Interfaces\TCP;

use App\Application\Service\ClientProxy\EventHandlerRegistryInterface;
use Buggregator\Client\Proto\Frame;
use Buggregator\Client\Proto\Server\Decoder;
use Psr\Log\LoggerInterface;
use Spiral\RoadRunner\Tcp\Request;
use Spiral\RoadRunner\Tcp\TcpWorkerInterface;
use Spiral\RoadRunnerBridge\Tcp\Response\CloseConnection;
use Spiral\RoadRunnerBridge\Tcp\Response\ContinueRead;
use Spiral\RoadRunnerBridge\Tcp\Response\ResponseInterface;
use Spiral\RoadRunnerBridge\Tcp\Service\ServiceInterface;

final class ClientProxyService implements ServiceInterface
{
    public function __construct(
        private readonly EventHandlerRegistryInterface $handlerRegistry,
        private readonly LoggerInterface $logger,
        private readonly Decoder $decoder,
    ) {
    }

    public function handle(Request $request): ResponseInterface
    {
        if ($request->event === TcpWorkerInterface::EVENT_CONNECTED) {
            return new ContinueRead();
        }

        $request = $this->decoder->decode(\trim($request->body));

        foreach ($request->getParsedPayload() as $event) {
            try {
                $this->handlePayload($event);
            } catch (\Throwable $e) {
                $this->logger->debug($e->getMessage());
            }
        }

        return new CloseConnection();
    }

    private function handlePayload(Frame $payload): void
    {
        $this->handlerRegistry->handle($payload);
    }
}
