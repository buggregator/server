<?php

declare(strict_types=1);

namespace Modules\Sentry\Interfaces\Http\Handler;

use App\Application\Commands\HandleReceivedEvent;
use App\Application\Event\EventType;
use App\Application\Service\HttpHandler\HandlerInterface;
use Modules\Sentry\Application\EventHandlerInterface;
use Modules\Sentry\Application\PayloadParser;
use Modules\Sentry\Application\SecretKeyValidator;
use Psr\Http\Message\ResponseInterface;
use Psr\Http\Message\ServerRequestInterface;
use Spiral\Core\Attribute\Singleton;
use Spiral\Cqrs\CommandBusInterface;
use Spiral\Http\Exception\ClientException\ForbiddenException;
use Spiral\Http\ResponseWrapper;

#[Singleton]
final readonly class EventHandler implements HandlerInterface
{
    public function __construct(
        private PayloadParser $payloadParser,
        private ResponseWrapper $responseWrapper,
        private EventHandlerInterface $handler,
        private CommandBusInterface $commands,
        private SecretKeyValidator $secretKeyValidator,
    ) {
    }

    public function priority(): int
    {
        return 0;
    }

    public function handle(ServerRequestInterface $request, \Closure $next): ResponseInterface
    {
        if (!$request->hasHeader('X-Sentry-Auth')) {
            return $next($request);
        }

        if (!$this->secretKeyValidator->validateRequest($request)) {
            throw new ForbiddenException('Invalid secret key');
        }

        $url = \rtrim($request->getUri()->getPath(), '/');
        $project = \explode('/', $url)[2] ?? null;

        $event = new EventType(type: 'sentry', project: $project);

        $payloads = $this->payloadParser->parse($request);

        match (true) {
            \str_ends_with($url, '/envelope') => $this->handleEnvelope($payloads, $event),
            \str_ends_with($url, '/store') => $this->handleEvent($payloads, $event),
            default => null,
        };

        return $this->responseWrapper->create(200);
    }

    private function handleEvent(array $data, EventType $eventType): void
    {
        $event = $this->handler->handle($data[0]);

        $this->commands->dispatch(
            new HandleReceivedEvent(
                type: 'sentry',
                payload: $event,
                project: $eventType->project,
            ),
        );
    }

    /**
     * TODO handle sentry transaction and session
     */
    private function handleEnvelope(array $data, EventType $eventType): void
    {
        if (\count($data) == 3) {
            match ($data[1]['type']) {
                'event' => $this->handleEvent([$data[2]], $eventType),
                default => null,
            };
        }
    }
}
