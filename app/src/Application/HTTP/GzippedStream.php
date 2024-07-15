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
     * @return string
     */
    public function getPayload(): string
    {
        return (string) $this->stream;
    }
}
