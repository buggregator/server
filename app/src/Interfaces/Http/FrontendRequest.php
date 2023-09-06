<?php

declare(strict_types=1);

namespace App\Interfaces\Http;

use App\Application\Service\HttpHandler\HandlerInterface;
use GuzzleHttp\Psr7\MimeType;
use Nyholm\Psr7\Response;
use Psr\Http\Message\ResponseInterface;
use Psr\Http\Message\ServerRequestInterface;
use Spiral\Http\Exception\ClientException\NotFoundException;

final class FrontendRequest implements HandlerInterface
{
    /**
     * @var array<non-empty-string, array{
     *     len: int<0, max>,
     *     content:non-empty-string,
     *     mime: non-empty-string
     * }>
     */
    private array $fileContent = [];

    public function __construct(
        private readonly string $publicPath,
    ) {
    }

    public function priority(): int
    {
        return 100_000;
    }

    public function handle(ServerRequestInterface $request, \Closure $next): ResponseInterface
    {
        if (!$this->isValidRequest($request)) {
            return $next($request);
        }

        $path = $request->getUri()->getPath();

        if ($path === '/') {
            $path = '/index.html';
        }

        $path = $this->publicPath . $path;

        if (!isset($this->fileContent[$path])) {
            if (!file_exists($path)) {
                throw new NotFoundException(\sprintf('File "%s" not found', $path));
            }

            $body = \file_get_contents($path);
            $this->fileContent[$path] = [
                'len' => \strlen($body),
                'content' => $body,
                'mime' => MimeType::fromFilename($path),
            ];
        }

        return new Response(
            200,
            [
                'Cache-Control' => 'public, max-age=300',
                'Content-Type' => $this->fileContent[$path]['mime'] . '; charset=utf-8',
                'Content-Length' => $this->fileContent[$path]['len'],
            ],
            $this->fileContent[$path]['content'],
        );
    }

    private function isValidRequest(ServerRequestInterface $request): bool
    {
        $path = $request->getUri()->getPath();

        return $path === '/' || \str_starts_with($path, '/_nuxt/') || $path === '/favicon/favicon.ico';
    }
}
