<?php

declare(strict_types=1);

namespace Modules\Sentry\Application;

use App\Application\HTTP\GzippedStreamFactory;
use Modules\Sentry\Application\DTO\JsonChunk;
use Modules\Sentry\Application\DTO\Payload;
use Modules\Sentry\Application\DTO\PayloadFactory;
use Psr\Http\Message\ServerRequestInterface;

final readonly class PayloadParser
{
    public function __construct(
        private GzippedStreamFactory $gzippedStreamFactory,
    ) {}

    public function parse(ServerRequestInterface $request): Payload
    {
        $isV4 = $request->getHeaderLine('Content-Type') === 'application/x-sentry-envelope' ||
            \str_contains($request->getHeaderLine('X-Sentry-Auth'), 'sentry_client=sentry.php');

        if (!$isV4) {
            throw new \InvalidArgumentException('Unsupported Sentry protocol version');
        }

        if ($request->getHeaderLine('Content-Encoding') === 'gzip') {
            return PayloadFactory::parseJson(
                $this->gzippedStreamFactory->createFromRequest($request)->getPayload(),
            );
        }

        return PayloadFactory::parseJson((string) $request->getBody());
    }
}
