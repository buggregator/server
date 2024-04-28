<?php

declare(strict_types=1);

namespace Modules\Inspector\Interfaces\Http\Handler;

use App\Application\Commands\HandleReceivedEvent;
use App\Application\Event\EventType;
use App\Application\Service\HttpHandler\HandlerInterface;
use Modules\Inspector\Application\SecretKeyValidator;
use Psr\Http\Message\ResponseInterface;
use Psr\Http\Message\ServerRequestInterface;
use Spiral\Cqrs\CommandBusInterface;
use Spiral\Http\Exception\ClientException;
use Spiral\Http\ResponseWrapper;

final readonly class EventHandler implements HandlerInterface
{
    public function __construct(
        private ResponseWrapper $responseWrapper,
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
        $eventType = $this->listenEvent($request);
        if ($eventType === null) {
            return $next($request);
        }

        if (!$this->secretKeyValidator->validateRequest($request)) {
            throw new ClientException\ForbiddenException('Invalid secret key');
        }

        $data = \json_decode(\base64_decode((string)$request->getBody()), true)
            ?? throw new ClientException\BadRequestException('Invalid data');

        $type = $data[0]['type'] ?? 'unknown';

        $data = match ($type) {
            'process',
            'request' => $data,
            default => throw new ClientException\BadRequestException(
                \sprintf('Invalid type "%s". [%s] expected.', $type, \implode(', ', ['process', 'request'])),
            ),
        };

        file_put_contents(directory('runtime') .'/inspectot.php', var_export($data, true));
        $this->commands->dispatch(
            new HandleReceivedEvent(type: $eventType->type, payload: $data, project: $eventType->project),
        );

        return $this->responseWrapper->create(200);
    }

    private function listenEvent(ServerRequestInterface $request): ?EventType
    {
        /** @var EventType|null $event */
        $event = $request->getAttribute('event');

        if ($event?->type === 'inspector') {
            return $event;
        }

        if (
            $request->hasHeader('X-Inspector-Key')
            || $request->hasHeader('X-Inspector-Version')
            || \str_ends_with((string)$request->getUri(), 'inspector')
        ) {
            return new EventType(type: 'inspector');
        }

        return null;
    }
}
