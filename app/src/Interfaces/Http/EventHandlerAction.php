<?php

declare(strict_types=1);

namespace App\Interfaces\Http;

use App\Application\Service\HttpHandler\CoreHandlerInterface;
use Psr\Http\Message\ResponseInterface;
use Psr\Http\Message\ServerRequestInterface;

final class EventHandlerAction
{
    public function handle(ServerRequestInterface $request, CoreHandlerInterface $handler): ResponseInterface
    {
        $auth = $request->getHeaderLine('Authorization');

        if (\str_starts_with($auth, 'Basic')) {
            $request = $request->withAttribute(
                'event-type',
                \rtrim(\base64_decode(\substr($auth, 6)), ':')
            );
        }

        return $handler->handle($request);
    }
}
