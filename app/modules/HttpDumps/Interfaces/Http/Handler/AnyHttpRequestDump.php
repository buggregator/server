<?php

declare(strict_types=1);

namespace Modules\HttpDumps\Interfaces\Http\Handler;

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
