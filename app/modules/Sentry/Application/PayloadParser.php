<?php

declare(strict_types=1);

namespace Modules\Sentry\Application;

use App\Application\HTTP\GzippedStreamFactory;
use Psr\Http\Message\ServerRequestInterface;

final readonly class PayloadParser
{
    public function __construct(
        private GzippedStreamFactory $gzippedStreamFactory,
    ) {}

    public function parse(ServerRequestInterface $request): array
    {
        $isV4 = $request->getHeaderLine('Content-Type') === 'application/x-sentry-envelope' ||
            \str_contains($request->getHeaderLine('X-Sentry-Auth'), 'sentry_client=sentry.php');

        if ($isV4) {
            if ($request->getHeaderLine('Content-Encoding') === 'gzip') {
                return \iterator_to_array($this->gzippedStreamFactory->createFromRequest($request)->getPayload());
            }

            $payloads = \explode("\n", (string) $request->getBody());

            return \array_map(
                static fn(string $payload): array => \json_decode($payload, true),
                \array_filter($payloads),
            );
        }

        return [$request->getParsedBody()];
    }
}
