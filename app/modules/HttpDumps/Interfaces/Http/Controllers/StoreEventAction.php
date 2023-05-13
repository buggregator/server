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
use Spiral\Storage\BucketInterface;
use Spiral\Storage\StorageInterface;

final class StoreEventAction
{
    private readonly BucketInterface $bucket;

    public function __construct(
        private readonly EventHandlerInterface $handler,
        StorageInterface $storage,
    ) {
        $this->bucket = $storage->bucket('attachments');
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
            new HandleReceivedEvent(type: 'http-dump', payload: $event)
        );
    }

    private function createPayload(ServerRequestInterface $request, string $uri): array
    {
        $fullUrl = (string)$request->getUri();

        $uri = \substr($fullUrl, \strpos($fullUrl, $uri));
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
                        $file = $this->bucket->write(
                            $filename = $id . '/' . $attachment->getClientFilename(),
                            $attachment->getStream()
                        );

                        return [
                            'id' => \md5($filename),
                            'name' => $attachment->getClientFilename(),
                            'uri' => $filename,
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
