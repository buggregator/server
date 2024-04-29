<?php

declare(strict_types=1);

namespace App\Application\HTTP\Middleware;

use App\Application\Event\EventType;
use Psr\Http\Message\ResponseInterface;
use Psr\Http\Message\ServerRequestInterface;
use Psr\Http\Server\MiddlewareInterface;
use Psr\Http\Server\RequestHandlerInterface;

final readonly class DetectEventTypeMiddleware implements MiddlewareInterface
{
    public function process(ServerRequestInterface $request, RequestHandlerInterface $handler): ResponseInterface
    {
        $detectors = [
            $this->detectFromUserInfo(...),
            $this->detectFromHeader(...),
            $this->detectFromBasicAuth(...),
        ];

        foreach ($detectors as $detector) {
            $info = $detector($request);
            if ($info !== null) {
                return $handler->handle(
                    $request->withAttribute('event', $info),
                );
            }
        }

        return $handler->handle(
            $request->withAttribute('event', new EventType(type: 'unknown', project: null)),
        );
    }

    private function detectFromBasicAuth(ServerRequestInterface $request): ?EventType
    {
        $auth = $request->getHeaderLine('Authorization');

        if (\str_starts_with($auth, 'Basic')) {
            [$type, $project] = $this->parseAuthString(\base64_decode(\substr($auth, 6)));

            return new EventType(
                type: $type,
                project: $project,
            );
        }

        return null;
    }

    private function detectFromUserInfo(ServerRequestInterface $request): ?EventType
    {
        $info = $request->getUri()->getUserInfo();

        if (empty($info)) {
            return null;
        }

        [$type, $project] = $this->parseAuthString($request->getUri()->getUserInfo());

        return new EventType(
            type: $type,
            project: empty($project) ? null : $project,
        );
    }

    private function detectFromHeader(ServerRequestInterface $request): ?EventType
    {
        $type = $request->getHeaderLine('X-Buggregator-Event');
        $project = $request->getHeaderLine('X-Buggregator-Project');

        if (empty($type)) {
            return null;
        }

        return new EventType(
            type: $type,
            project: empty($project) ? null : $project,
        );
    }

    private function parseAuthString(string $string): array
    {
        if (\str_contains($string, ':')) {
            return \explode(':', $string, 2);
        }

        return [$string, null];
    }
}
