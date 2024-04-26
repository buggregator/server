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
        return $handler->handle($request);
    }
}
