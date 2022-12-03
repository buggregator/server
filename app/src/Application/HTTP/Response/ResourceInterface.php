<?php

declare(strict_types=1);

namespace App\Application\HTTP\Response;

use Psr\Http\Message\ResponseInterface;
use Psr\Http\Message\ServerRequestInterface;

interface ResourceInterface
{
    public function resolve(ServerRequestInterface $request): array;

    public function toResponse(ServerRequestInterface $request, ResponseInterface $response): ResponseInterface;
}
