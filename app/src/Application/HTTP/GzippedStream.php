<?php

declare(strict_types=1);

namespace App\Application\HTTP;

use Psr\Http\Message\StreamInterface;

final readonly class GzippedStream
{
    public function __construct(
        private StreamInterface $stream,
    ) {}

    /**
     * @return iterable<array|string>
     */
    public function getPayload(): iterable
    {
        $payloads = \array_filter(\explode("\n", (string) $this->stream));

        foreach ($payloads as $payload) {
            if (!\json_validate($payload)) {
                yield $payload;
                continue;
            }

            yield \json_decode($payload, true, 512, \JSON_THROW_ON_ERROR);
        }
    }
}
