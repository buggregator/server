<?php

declare(strict_types=1);

namespace App\Application\OAuth;

use Psr\Http\Message\UriInterface;
use Spiral\Http\Request\InputManager;

interface AuthProviderInterface
{
    public function getLoginUrl(): UriInterface;

    public function isAuthenticated(): bool;

    public function getUser(): ?User;

    public function authenticate(InputManager $input): void;

    public function getLogoutUrl(): ?UriInterface;

    public function logout(): void;
}
