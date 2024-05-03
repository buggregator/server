<?php

declare(strict_types=1);

namespace Tests\App\Smtp;

use Modules\Smtp\Interfaces\TCP\Service as SmtpService;
use Spiral\RoadRunner\Tcp\Request;
use Spiral\RoadRunner\Tcp\TcpEvent;
use Spiral\RoadRunnerBridge\Tcp\Response\ResponseInterface;
use Symfony\Component\Mailer\Transport\Smtp\Stream\AbstractStream;

final class FakeStream extends AbstractStream
{
    public function __construct(
        private readonly SmtpService $service,
        private readonly string $uuid,
    ) {
        $this->out = \fopen('php://memory', 'w+');
    }

    public function getUuid(): string
    {
        return $this->uuid;
    }

    public function initialize(): void
    {
        $this->err = \fopen('php://memory', 'w+');

        $this->response[] = $this->sendRequest(
            $this->buildRequest('', TcpEvent::Connected),
        );
    }

    public function write(string $bytes, bool $debug = true): void
    {
        $event = TcpEvent::Data;

        $response = $this->sendRequest($this->buildRequest($bytes, $event));
        $this->response[] = $response;
    }

    public function flush(): void {}

    public function isTLS(): bool
    {
        return true;
    }

    public function setTimeout(float $timeout): self
    {
        return $this;
    }

    public function setHost(string $host): self
    {
        return $this;
    }

    public function setPort(int $port): self
    {
        return $this;
    }

    public function setStreamOptions(array $options): self
    {
        return $this;
    }

    public function setSourceIp(string $ip): self
    {
        return $this;
    }

    public function disableTls(): self
    {
        return $this;
    }

    protected function getReadConnectionDescription(): string
    {
        return 'fake';
    }

    private function buildRequest(string $message, TcpEvent $event = TcpEvent::Data): Request
    {
        return new Request(
            remoteAddr: '127.0.0.1',
            event: $event,
            body: $message,
            connectionUuid: $this->uuid,
            server: 'localhost',
        );
    }

    private function sendRequest(Request $request): ResponseInterface
    {
        $response = $this->service->handle($request);

        \fclose($this->out);
        $this->out = \fopen('php://memory', 'w+');
        \fwrite($this->out, $response->getBody());
        \rewind($this->out);

        return $response;
    }

}
