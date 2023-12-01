<?php

declare(strict_types=1);

namespace Modules\HttpDumps\Interfaces\Http\Handler;

use App\Application\Commands\HandleReceivedEvent;
use App\Application\Service\HttpHandler\HandlerInterface;
use Carbon\Carbon;
use Modules\HttpDumps\Application\EventHandlerInterface;
use Psr\Http\Message\ResponseInterface;
use Psr\Http\Message\ServerRequestInterface;
use Psr\Http\Message\UploadedFileInterface;
use Spiral\Cqrs\CommandBusInterface;
use Spiral\Http\ResponseWrapper;
use Spiral\Storage\BucketInterface;
use Spiral\Storage\StorageInterface;

final class AnyHttpRequestDump implements HandlerInterface
{
    private readonly BucketInterface $bucket;

    public function __construct(
        private readonly CommandBusInterface $commands,
        private readonly EventHandlerInterface $handler,
        private readonly ResponseWrapper $responseWrapper,
        StorageInterface $storage,
    ) {
        $this->bucket = $storage->bucket('attachments');
    }

    public function priority(): int
    {
        return 0;
    }

    private function isValidRequest(ServerRequestInterface $request): bool
    {
        return $request->getHeaderLine('X-Buggregator-Event') === 'http-dump'
            || $request->getAttribute('event-type') === 'http-dump';
    }

    public function handle(ServerRequestInterface $request, \Closure $next): ResponseInterface
    {
        if (!$this->isValidRequest($request)) {
            return $next($request);
        }

        $payload = $this->createPayload($request);

        $event = $this->handler->handle($payload);

        $this->commands->dispatch(
            new HandleReceivedEvent(type: 'http-dump', payload: $event),
        );

        return $this->responseWrapper->create(200);
    }

    private function createPayload(ServerRequestInterface $request): array
    {
        $uri = \ltrim($request->getUri()->getPath(), '/');
        $id = \md5(Carbon::now()->toDateTimeString());

        return [
            'received_at' => Carbon::now()->toDateTimeString(),
            'host' => $request->getHeaderLine('Host'),
            'request' => [
                'method' => $request->getMethod(),
                'uri' => $uri,
                'headers' => $request->getHeaders(),
                'body' => (string)$request->getBody(),
                'query' => $request->getQueryParams(),
                'post' => $request->getParsedBody() ?? [],
                'cookies' => $request->getCookieParams(),
                'files' => \array_map(
                    function (UploadedFileInterface $attachment) use ($id) {
                        $this->bucket->write(
                            $filename = $id . '/' . $attachment->getClientFilename(),
                            $attachment->getStream(),
                        );

                        return [
                            'id' => \md5($filename),
                            'name' => $attachment->getClientFilename(),
                            'uri' => $filename,
                            'size' => $attachment->getSize(),
                            'mime' => $attachment->getClientMediaType(),
                        ];
                    },
                    $request->getUploadedFiles(),
                ),
            ],
        ];
    }
}
