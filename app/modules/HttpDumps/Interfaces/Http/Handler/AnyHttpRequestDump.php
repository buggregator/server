<?php

declare(strict_types=1);

namespace Modules\HttpDumps\Interfaces\Http\Handler;

use App\Application\Commands\HandleReceivedEvent;
use App\Application\Domain\ValueObjects\Uuid;
use App\Application\Event\EventType;
use App\Application\Service\HttpHandler\HandlerInterface;
use Carbon\Carbon;
use Modules\HttpDumps\Application\EventHandlerInterface;
use Modules\HttpDumps\Domain\Attachment;
use Modules\HttpDumps\Domain\AttachmentStorageInterface;
use Psr\Container\ContainerInterface;
use Psr\Http\Message\ResponseInterface;
use Psr\Http\Message\ServerRequestInterface;
use Spiral\Cqrs\CommandBusInterface;
use Spiral\Http\ResponseWrapper;

final readonly class AnyHttpRequestDump implements HandlerInterface
{
    public function __construct(
        private CommandBusInterface $commands,
        private EventHandlerInterface $handler,
        private ResponseWrapper $responseWrapper,
        private ContainerInterface $container,
    ) {}

    public function priority(): int
    {
        return 0;
    }

    public function handle(ServerRequestInterface $request, \Closure $next): ResponseInterface
    {
        $eventType = $this->listenEvent($request);

        if (!$eventType instanceof EventType) {
            return $next($request);
        }

        $payload = $this->createPayload($request);

        $event = $this->handler->handle($payload);

        $this->commands->dispatch(
            new HandleReceivedEvent(type: $eventType->type, payload: $event, project: $eventType->project),
        );

        return $this->responseWrapper->create(200);
    }

    private function createPayload(ServerRequestInterface $request): array
    {
        $uri = \ltrim($request->getUri()->getPath(), '/');

        $uuid = Uuid::generate();
        $result = $this->container->get(AttachmentStorageInterface::class)
            ->store(eventUuid: $uuid, attachments: $request->getUploadedFiles());

        return [
            'received_at' => Carbon::now()->toDateTimeString(),
            'host' => $request->getHeaderLine('Host'),
            'request' => [
                'method' => $request->getMethod(),
                'uri' => $uri,
                'headers' => $request->getHeaders(),
                'body' => (string) $request->getBody(),
                'query' => $request->getQueryParams(),
                'post' => $request->getParsedBody() ?? [],
                'cookies' => $request->getCookieParams(),
                'files' => \array_map(
                    fn(Attachment $attachment) => [
                        'uuid' => (string) $attachment->getUuid(),
                        'name' => $attachment->getFilename(),
                        'size' => $attachment->getSize(),
                        'mime' => $attachment->getMime(),
                    ],
                    $result,
                ),
            ],
        ];
    }

    private function listenEvent(ServerRequestInterface $request): ?EventType
    {
        /** @var EventType|null $event */
        $event = $request->getAttribute('event');

        if ($event?->type === 'http-dump') {
            return $event;
        }

        return null;
    }
}
