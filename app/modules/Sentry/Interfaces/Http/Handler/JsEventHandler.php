<?php

declare(strict_types=1);

namespace Modules\Sentry\Interfaces\Http\Handler;

use App\Application\Event\EventType;
use App\Application\Service\HttpHandler\HandlerInterface;
use Modules\Sentry\Application\DTO\PayloadFactory;
use Modules\Sentry\Application\EventHandlerInterface;
use Modules\Sentry\Application\SecretKeyValidator;
use Psr\Http\Message\ResponseInterface;
use Psr\Http\Message\ServerRequestInterface;
use Spiral\Http\Exception\ClientException\ForbiddenException;
use Spiral\Http\ResponseWrapper;

final readonly class JsEventHandler implements HandlerInterface
{
    public function __construct(
        private ResponseWrapper $responseWrapper,
        private EventHandlerInterface $handler,
        private SecretKeyValidator $secretKeyValidator,
    ) {}

    public function priority(): int
    {
        return 1;
    }

    // TODO: add support for sentry project
    public function handle(ServerRequestInterface $request, \Closure $next): ResponseInterface
    {
        if (!$this->isValidRequest($request)) {
            return $next($request);
        }

        if (!$this->secretKeyValidator->validateRequest($request)) {
            throw new ForbiddenException('Invalid secret key');
        }

        $url = \rtrim($request->getUri()->getPath(), '/');
        $project = \explode('/', $url)[2] ?? null;

        $payload = PayloadFactory::parseJson((string) $request->getBody());

        $this->handler->handle($payload, new EventType(type: 'sentry', project: $project));

        return $this->responseWrapper->create(200);
    }

    private function isValidRequest(ServerRequestInterface $request): bool
    {
        return isset($request->getQueryParams()['sentry_key']);
    }
}
