<?php

declare(strict_types=1);

namespace Modules\Sentry\Application;

use App\Application\HTTP\GzippedStreamFactory;
use Modules\Sentry\Application\DTO\BlobChunk;
use Modules\Sentry\Application\DTO\JsonChunk;
use Modules\Sentry\Application\DTO\Payload;
use Modules\Sentry\Application\DTO\TypeChunk;
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

        if ($isV4) {
            if ($request->getHeaderLine('Content-Encoding') === 'gzip') {
                $chunks = [];

                foreach ($this->gzippedStreamFactory->createFromRequest($request)->getPayload() as $payload) {
                    if (\is_string($payload)) {
                        $chunks[] = new BlobChunk($payload);
                        continue;
                    }

                    if (isset($payload['type'])) {
                        $chunks[] = new TypeChunk($payload);
                        continue;
                    }

                    $chunks[] = new JsonChunk($payload);
                }

                return new Payload($chunks);
            }

            return Payload::parse((string) $request->getBody());
        }

        return new Payload(
            [new JsonChunk($request->getParsedBody())],
        );
    }
}
