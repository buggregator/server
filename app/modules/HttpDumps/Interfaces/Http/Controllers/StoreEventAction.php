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

    #[Route(route: 'httpdump[/<uri:.*>]', name: 'httpdump.event.store', group: 'api', priority: 100)]
    public function __invoke(
        ServerRequestInterface $request,
        CommandBusInterface $commands,
        string $uri = '/',
    ): void {
        $event = $this->handler->handle(
            $this->createPayload($request, $uri)
        );

        $commands->dispatch(
            new HandleReceivedEvent(type: 'httpdump', payload: $event)
        );
    }

    private function createPayload(ServerRequestInterface $request, string $uri): array
    {
        return [
            'received_at' => Carbon::now()->toDateTimeString(),
            'request' => [
                'method' => $request->getMethod(),
                'uri' => $uri,
                'headers' => $request->getHeaders(),
                'body' => (string)$request->getBody(),
                'query' => $request->getQueryParams(),
                'post' => $request->getParsedBody(),
                'attributes' => $request->getAttributes(),
                'server' => $request->getServerParams(),
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
