<?php

declare(strict_types=1);

namespace App\Application\Service\ClientProxy;

use App\Application\Service\HttpHandler\CoreHandlerInterface;
use Buggregator\Client\Proto\Frame;

final class HttpRequestsHandler implements EventHandlerInterface
{
    public function __construct(
        private readonly CoreHandlerInterface $handler,
    ) {
    }

    public function isSupported(Frame $frame): bool
    {
        return $frame instanceof Frame\Http;
    }

    /**
     * @param Frame\Http $frame
     */
    public function handle(Frame $frame): void
    {
        $request = $frame->request;

        $auth = (string)$request->getHeaderLine('Authorization');

        if (\str_starts_with($auth, 'Basic')) {
            $request = $request->withAttribute(
                'event-type',
                \rtrim(\base64_decode(\substr($auth, 6)), ':')
            );
        }

        $this->handler->handle($request);
    }
}
