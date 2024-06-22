<?php

declare(strict_types=1);

namespace Modules\Sentry\Interfaces\Http\Handler;

use App\Application\Event\EventType;
use App\Application\Service\HttpHandler\HandlerInterface;
use Modules\Sentry\Application\EventHandlerInterface;
use Modules\Sentry\Application\PayloadParser;
use Modules\Sentry\Application\SecretKeyValidator;
use Psr\Http\Message\ResponseInterface;
use Psr\Http\Message\ServerRequestInterface;
use Spiral\Core\Attribute\Singleton;
use Spiral\Http\Exception\ClientException\ForbiddenException;
use Spiral\Http\ResponseWrapper;

#[Singleton]
final readonly class EventHandler implements HandlerInterface
{
    public function __construct(
        private PayloadParser $payloadParser,
        private ResponseWrapper $responseWrapper,
        private EventHandlerInterface $handler,
        private SecretKeyValidator $secretKeyValidator,
    ) {}

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

        $payload = $this->payloadParser->parse($request);
        $this->handler->handle($payload, new EventType(type: 'sentry', project: $project));

        return $this->responseWrapper->create(200);
    }
}
