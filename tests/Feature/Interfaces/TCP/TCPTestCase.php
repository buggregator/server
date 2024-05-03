<?php

declare(strict_types=1);

namespace Tests\Feature\Interfaces\TCP;

use Modules\Monolog\Interfaces\TCP\Service as MonologService;
use Modules\VarDumper\Interfaces\TCP\Service as VarDumperService;
use Modules\Smtp\Interfaces\TCP\Service as SmtpService;
use Ramsey\Uuid\Uuid;
use Ramsey\Uuid\UuidInterface;
use Spiral\RoadRunner\Tcp\Request;
use Spiral\RoadRunner\Tcp\TcpEvent;
use Spiral\RoadRunnerBridge\Tcp\Response\ResponseInterface;
use Symfony\Component\Mailer\Transport\Smtp\EsmtpTransport;
use Tests\App\Smtp\FakeStream;
use Tests\DatabaseTestCase;

abstract class TCPTestCase extends DatabaseTestCase
{
    public function handleMonologRequest(string $message): ResponseInterface
    {
        return $this
            ->get(MonologService::class)
            ->handle($this->buildRequest(message: $message));
    }

    public function handleVarDumperRequest(string $message): ResponseInterface
    {
        return $this
            ->get(VarDumperService::class)
            ->handle($this->buildRequest(message: $message));
    }

    public function handleSmtpRequest(string $message, TcpEvent $event = TcpEvent::Data): ResponseInterface
    {
        return $this
            ->get(SmtpService::class)
            ->handle($this->buildRequest(message: $message, event: $event));
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

    protected function buildSmtpClient(string $username = 'homestead', ?UuidInterface $uuid = null): EsmtpTransport
    {
        $client = new EsmtpTransport(
            stream: new FakeStream(
                service: $this->get(SmtpService::class),
                uuid: (string) $uuid ?? Uuid::uuid7(),
            ),
        );

        $client->setUsername($username);
        $client->setPassword('password');

        return $client;
    }
}
