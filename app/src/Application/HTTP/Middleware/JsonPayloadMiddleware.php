<?php

declare(strict_types=1);

namespace App\Application\HTTP\Middleware;

use Psr\Http\Message\ResponseInterface;
use Psr\Http\Message\ServerRequestInterface;
use Psr\Http\Server\MiddlewareInterface;
use Psr\Http\Server\RequestHandlerInterface;
use Spiral\Bootloader\Http\JsonPayloadConfig;

final readonly class JsonPayloadMiddleware implements MiddlewareInterface
{
    public function __construct(
        private JsonPayloadConfig $config,
    ) {}

    public function process(ServerRequestInterface $request, RequestHandlerInterface $handler): ResponseInterface
    {
        if ($this->isJsonPayload($request)) {
            $body = (string) $request->getBody();
            if ($body !== '') {
                $parsedBody = \json_decode($body, true);
                if (\json_last_error() === 0) {
                    $request = $request->withParsedBody($parsedBody);
                }
            }
        }

        return $handler->handle($request);
    }

    private function isJsonPayload(ServerRequestInterface $request): bool
    {
        if ($request->getHeaderLine('Content-Encoding') === 'gzip') {
            return false;
        }

        $contentType = $request->getHeaderLine('Content-Type');

        foreach ($this->config->getContentTypes() as $allowedType) {
            if (\stripos($contentType, $allowedType) === 0) {
                return true;
            }
        }

        return false;
    }
}
