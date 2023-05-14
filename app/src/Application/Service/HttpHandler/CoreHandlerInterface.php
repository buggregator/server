<?php

declare(strict_types=1);

namespace App\Application\Service\HttpHandler;

use Psr\Http\Message\ResponseInterface;
use Psr\Http\Message\ServerRequestInterface;

interface CoreHandlerInterface
{
    public function handle(ServerRequestInterface $request): ResponseInterface;
}
