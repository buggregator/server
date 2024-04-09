<?php

declare(strict_types=1);

namespace Modules\Smtp\Interfaces\TCP;

use App\Application\Commands\HandleReceivedEvent;
use Carbon\Carbon;
use Modules\Smtp\Application\Mail\Parser;
use Psr\SimpleCache\CacheInterface;
use Spiral\Cache\CacheStorageProviderInterface;
use Spiral\Cqrs\CommandBusInterface;
use Spiral\RoadRunner\Tcp\Request;
use Spiral\RoadRunner\Tcp\TcpEvent;
use Spiral\RoadRunner\Tcp\TcpResponse;
use Spiral\RoadRunnerBridge\Tcp\Response\CloseConnection;
use Spiral\RoadRunnerBridge\Tcp\Response\RespondMessage;
use Spiral\RoadRunnerBridge\Tcp\Response\ResponseInterface;
use Spiral\RoadRunnerBridge\Tcp\Service\ServiceInterface;
use Spiral\Storage\StorageInterface;

final class Service implements ServiceInterface
{
    private const READY = 220;
    public const OK = 250;
    public const CLOSING = 221;
    public const START_MAIL_INPUT = 354;

    private readonly CacheInterface $cache;

    public function __construct(
        private readonly CommandBusInterface $commands,
        private readonly StorageInterface $storage,
        CacheStorageProviderInterface $provider,
    ) {
        $this->cache = $provider->storage('local');
    }

    public function handle(Request $request): ResponseInterface
    {
        if ($request->event === TcpEvent::Connected) {
            return $this->send(self::READY, 'mailamie');
        }

        $cacheKey = 'smtp:' . $request->connectionUuid;
        $message = $this->cache->get($cacheKey, []);

        $response = new CloseConnection();
        $dispatched = false;

        if ($request->event === TcpEvent::Close) {
            $this->cache->delete($cacheKey);

            return new CloseConnection();
        } elseif (\preg_match('/^(EHLO|HELO|MAIL FROM:)/', $request->body)) {
            $response = $this->send(self::OK);
        } elseif (\preg_match('/^RCPT TO:<(.*)>/', $request->body, $matches)) {
            $message['recipients'][] = $matches[0];
            $response = $this->send(self::OK);
        } elseif (\str_starts_with($request->body, 'QUIT')) {
            $response = $this->send(self::CLOSING, null, true);
        } elseif ($request->body === "DATA\r\n") {
            $response = $this->send(self::START_MAIL_INPUT);
            $message['collecting'] = true;
        } elseif ($message['collecting'] ?? false) {
            $content = $message['content'] ?? '';
            $response = $this->send(self::OK);
            $content .= \preg_replace("/^(\.\.)/m", '.', $request->body);

            if ($this->endOfContentDetected($request->body)) {
                $messages = \array_filter(\explode("\r\n.\r\n", $content));

                if (\count($messages) === 1) {
                    $this->dispatchMessage($content);
                    $this->cache->delete($cacheKey);
                    $dispatched = true;
                } elseif (!empty($messages[1])) {
                    $this->dispatchMessage($messages[0]);
                    $this->cache->delete($cacheKey);
                    $dispatched = true;
                }
            }

            $message['content'] = $content;
        }

        if (
            $response instanceof CloseConnection ||
            $response->getAction() === TcpResponse::RespondClose ||
            $dispatched
        ) {
            $this->cache->delete($cacheKey);
            return $response;
        }

        $this->cache->set(
            $cacheKey,
            $message,
            Carbon::now()->addMinutes(5)->diffAsCarbonInterval(),
        );

        return $response;
    }

    private function dispatchMessage(string $message): void
    {
        $data = (new Parser($this->storage))
            ->parse($message)
            ->storeAttachments()
            ->jsonSerialize();

        $this->commands->dispatch(
            new HandleReceivedEvent(type: 'smtp', payload: $data),
        );
    }

    private function endOfContentDetected(string $data): bool
    {
        return \str_ends_with($data, "\r\n.\r\n");
    }

    private function send(int $statusCode, string|null $comment = null, bool $close = false): RespondMessage
    {
        $response = \implode(' ', \array_filter([$statusCode, $comment]));

        return new RespondMessage("{$response} \r\n", $close);
    }
}
