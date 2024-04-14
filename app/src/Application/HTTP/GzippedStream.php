<?php

declare(strict_types=1);

namespace App\Application\HTTP;

use Psr\Http\Message\StreamInterface;

final readonly class GzippedStream
{
    public function __construct(
        private StreamInterface $stream,
    ) {}

    public function getPayload(): \Traversable
    {
        $payloads = \array_filter(\explode("\n", (string) $this->stream));

        foreach ($payloads as $payload) {
            yield \json_decode($payload, true);
        }
    }
}
