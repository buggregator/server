<?php

declare(strict_types=1);

namespace Modules\HttpDumps\Interfaces\Http\Handler;

use App\Application\Commands\HandleReceivedEvent;
use App\Application\Commands\StoreAttachment;
use App\Application\Service\HttpHandler\HandlerInterface;
use Carbon\Carbon;
use Modules\HttpDumps\Application\EventHandlerInterface;
use Psr\Http\Message\ResponseInterface;
use Psr\Http\Message\ServerRequestInterface;
use Psr\Http\Message\UploadedFileInterface;
use Spiral\Cqrs\CommandBusInterface;
use Spiral\Http\ResponseWrapper;
use Spiral\Storage\FileInterface;

final class AnyHttpRequestDump implements HandlerInterface
{
    public function __construct(
        private readonly CommandBusInterface $commands,
        private readonly EventHandlerInterface $handler,
        private readonly ResponseWrapper $responseWrapper,
    ) {
    }

    public function priority(): int
    {
        return 100_000;
    }

    public function handle(ServerRequestInterface $request, \Closure $next): ResponseInterface
    {
        $payload = $this->createPayload($request);

        $event = $this->handler->handle($payload);

        $this->commands->dispatch(
            new HandleReceivedEvent(type: 'http-dump', payload: $event)
        );

        return $this->responseWrapper->create(200);
    }

    private function createPayload(ServerRequestInterface $request): array
    {
        $uri = \ltrim($request->getUri()->getPath(), '/');

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
                    function (UploadedFileInterface $attachment): array {
                        /** @var FileInterface $file */
                        $file = $this->commands->dispatch(
                            new StoreAttachment(
                                type: 'http-dump',
                                filename: $attachment->getClientFilename(),
                                content: $attachment->getStream(),
                            )
                        );

                        return [
                            'id' => \md5($file->getPathname()),
                            'name' => $attachment->getClientFilename(),
                            'uri' => $file->getPathname(),
                            'size' => $attachment->getSize(),
                            'mime' => $attachment->getClientMediaType(),
                        ];
                    },
                    $request->getUploadedFiles()
                ),
            ]
        ];
    }
}
