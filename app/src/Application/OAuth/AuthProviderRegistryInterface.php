<?php

declare(strict_types=1);

namespace App\Application\OAuth;

use App\Application\Exception\AuthProviderNotFound;

interface AuthProviderRegistryInterface
{
    /**
     * @param class-string<AuthProviderInterface> $provider
     */
    public function register(string $name, string $provider): void;

    /**
     * @throws AuthProviderNotFound
     */
    public function get(string $name): AuthProviderInterface;
}
