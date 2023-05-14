<?php

declare(strict_types=1);

namespace Modules\Inspector\Interfaces\Http\Handler;

use App\Application\Commands\HandleReceivedEvent;
use App\Application\Service\HttpHandler\HandlerInterface;
use Psr\Http\Message\ResponseInterface;
use Psr\Http\Message\ServerRequestInterface;
use Spiral\Cqrs\CommandBusInterface;
use Spiral\Http\Exception\ClientException;
use Spiral\Http\ResponseWrapper;

final class EventHandler implements HandlerInterface
{
    public function __construct(
        private readonly ResponseWrapper $responseWrapper,
        private readonly CommandBusInterface $commands,
    ) {
    }

    public function priority(): int
    {
        return 0;
    }

    public function handle(ServerRequestInterface $request, \Closure $next): ResponseInterface
    {
        if (!$this->isValidRequest($request)) {
            return $next($request);
        }

        $data = \json_decode(\base64_decode((string)$request->getBody()), true)
            ?? throw new ClientException\BadRequestException('Invalid data');

        $type = $data[0]['type'] ?? 'unknown';
        if ($type !== 'request') {
            throw new ClientException\BadRequestException('Invalid data');
        }

        $this->commands->dispatch(
            new HandleReceivedEvent(type: 'inspector', payload: $data)
        );

        return $this->responseWrapper->create(200);
    }

    private function isValidRequest(ServerRequestInterface $request): bool
    {
        return $request->getHeaderLine('X-Buggregator-Event') === 'inspector'
            || $request->getAttribute('event-type') === 'inspector'
            || $request->getUri()->getUserInfo() === 'inspector'
            || $request->hasHeader('X-Inspector-Key')
            || $request->hasHeader('X-Inspector-Version')
            || \str_ends_with((string)$request->getUri(), 'inspector');
    }
}
