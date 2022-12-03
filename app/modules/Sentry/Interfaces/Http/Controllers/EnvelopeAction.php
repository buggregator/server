<?php

declare(strict_types=1);

namespace Modules\Sentry\Interfaces\Http\Controllers;

use Http\Message\Encoding\GzipDecodeStream;
use Psr\Http\Message\ServerRequestInterface;
use Spiral\Router\Annotation\Route;

final class EnvelopeAction
{
    #[Route(route: 'api/<projectId>/envelope', name: 'sentry.event.envelope', methods: ['POST'], group: 'api')]
    public function __invoke(int $projectId, ServerRequestInterface $request): void
    {
        $content = (new GzipDecodeStream($request->getBody()))->getContents();

        $data = \array_map(
            fn(string $line) => \json_decode($line, true),
            \array_filter(\explode("\n", $content))
        );

        if (\count($data) == 3) {
            match ($data[1]['type']) {
                'transaction' => $this->handleTransaction($data),
                'session' => $this->handleSession($data),
            };
        }
    }

    private function handleTransaction(array $data): void
    {
    }

    private function handleSession(array $data): void
    {
    }
}
