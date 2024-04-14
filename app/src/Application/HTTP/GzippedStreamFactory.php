<?php

declare(strict_types=1);

namespace App\Application\HTTP;

use GuzzleHttp\Psr7\Stream;
use Http\Message\Encoding\GzipDecodeStream;
use Psr\Http\Message\ServerRequestInterface;

final class GzippedStreamFactory
{
    public function createFromRequest(ServerRequestInterface $request): GzippedStream
    {
        $content = (string) $request->getBody();

        $resource = fopen('php://temp', 'r+');
        fwrite($resource, $content);
        rewind($resource);

        return new GzippedStream(
            new GzipDecodeStream(new Stream($resource))
        );
    }
}
