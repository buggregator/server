<?php

declare(strict_types=1);

namespace Modules\HttpDumps\Interfaces\Http\Controllers;

use App\Application\Commands\HandleReceivedEvent;
use Carbon\Carbon;
use Modules\HttpDumps\Application\EventHandlerInterface;
use Psr\Http\Message\ServerRequestInterface;
use Psr\Http\Message\UploadedFileInterface;
use Spiral\Cqrs\CommandBusInterface;
use Spiral\Router\Annotation\Route;

final class StoreEventAction
{
    public function __construct(
        private readonly EventHandlerInterface $handler
    ) {
    }

    #[Route(route: 'http-dumps[/<uri:.*>]', name: 'http-dumps.event.store', group: 'api', priority: 100)]
    public function __invoke(
        ServerRequestInterface $request,
        CommandBusInterface $commands,
        string $uri = '/',
    ): void {
        $payload = $this->createPayload($request, $uri);

        $event = $this->handler->handle($payload);

        $commands->dispatch(
            new HandleReceivedEvent(type: 'httpdump', payload: $event)
        );
    }

    private function createPayload(ServerRequestInterface $request, string $uri): array
    {
        $fullUrl = (string)$request->getUri();

        $uri = \substr($fullUrl, \strpos($fullUrl, $uri));
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
                    static fn(UploadedFileInterface $file) => [
                        'originalName' => $file->getClientFilename(),
                        'mime' => $file->getClientMediaType(),
                        'size' => $file->getSize(),
                    ],
                    $request->getUploadedFiles()
                ),
            ]
        ];
    }
}
