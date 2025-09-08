<?php

declare(strict_types=1);

namespace App\Interfaces\Http\Handler;

use App\Application\Service\HttpHandler\HandlerInterface;
use GuzzleHttp\Psr7\MimeType;
use Nyholm\Psr7\Response;
use Psr\Http\Message\ResponseInterface;
use Psr\Http\Message\ServerRequestInterface;

final class FrontendRequest implements HandlerInterface
{
    /** @var array<string, Content> */
    private array $fileContent = [];

    public function __construct(
        private readonly string $publicPath,
    ) {}

    public function priority(): int
    {
        return 100_000;
    }

    public function handle(ServerRequestInterface $request, \Closure $next): ResponseInterface
    {
        if (!$this->isValidRequest($request)) {
            return $next($request);
        }

        /** @var non-empty-string $path */
        $path = $request->getUri()->getPath();

        if ($path === '/') {
            $path = '/index.html';
        }

        $path = $this->publicPath . $path;

        if (!isset($this->fileContent[$path])) {
            if (!file_exists($path)) {
                // Similar to Nginx's retry-files functionality.
                $path = $this->publicPath . '/index.html';
            }

            $body = \file_get_contents($path);
            $this->fileContent[$path] = new Content(
                content: $body,
                mime: MimeType::fromFilename($path) ?? 'text/plain',
            );
        }

        return new Response(
            200,
            [
                'Cache-Control' => 'public, max-age=300',
                'Content-Type' => $this->fileContent[$path]->contentType,
                'Content-Length' => $this->fileContent[$path]->len,
            ],
            (string) $this->fileContent[$path],
        );
    }

    private function isValidRequest(ServerRequestInterface $request): bool
    {
        $path = $request->getUri()->getPath();

        $frontendRoutes = [
            '/',
            '/ray',
            '/smtp',
            '/sentry',
            '/monolog',
            '/profiler',
            '/settings',
            '/var-dump',
            '/http-dump',
            '/inspector',
        ];

        return in_array($path, $frontendRoutes)
            || \str_starts_with($path, '/src/')
            || \str_starts_with($path, '/assets/')
            || $path === '/favicon/favicon.ico'
            || $path === '/bg.jpg';
    }
}
