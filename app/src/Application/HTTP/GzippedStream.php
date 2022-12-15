<?php

declare(strict_types=1);

namespace App\Application\HTTP;

use Psr\Http\Message\StreamInterface;

final class GzippedStream
{
    public function __construct(
        private readonly StreamInterface $stream,
    ) {
    }

    public function getPayload(): array
    {
        return \json_decode($this->stream->getContents(), true);
    }

    public function getEnvelopePayload(): array
    {
        return \array_map(
            fn(string $line): array => \json_decode($line, true),
            \array_filter(\explode("\n", $this->stream->getContents()))
        );
    }
}
