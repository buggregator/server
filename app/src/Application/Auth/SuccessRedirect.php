<?php

declare(strict_types=1);

namespace App\Application\Auth;

use Psr\Http\Message\ResponseInterface;
use Psr\Http\Message\UriInterface;
use Spiral\Http\ResponseWrapper;

final readonly class SuccessRedirect
{
    public function __construct(
        private ResponseWrapper $response,
        private UriInterface $redirectUrl,
    ) {}

    public function makeResponse(string $token): ResponseInterface
    {
        return $this->response->redirect($this->redirectUrl . '?token=' . $token);
    }
}
