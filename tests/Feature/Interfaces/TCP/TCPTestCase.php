<?php

declare(strict_types=1);

namespace Tests\Feature\Interfaces\TCP;

use Modules\Monolog\Interfaces\TCP\Service as MonologService;
use Spiral\RoadRunner\Tcp\Request;
use Spiral\RoadRunner\Tcp\TcpEvent;
use Spiral\RoadRunnerBridge\Tcp\Response\ResponseInterface;
use Tests\DatabaseTestCase;

abstract class TCPTestCase extends DatabaseTestCase
{
    public function handleMonologRequest(string $message): ResponseInterface
    {
        return $this
            ->get(MonologService::class)
            ->handle($this->buildRequest(message: $message));
    }

    private function buildRequest(string $message, TcpEvent $event = TcpEvent::Data): Request
    {
        return new Request(
            remoteAddr: '127.0.0.1',
            event: $event,
            body: $message,
            connectionUuid: '018f2586-4be9-7168-942e-0ce0c104961',
            server: 'localhost',
        );
    }
}
