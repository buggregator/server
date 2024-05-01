<?php

declare(strict_types=1);

namespace App\Application\OAuth;

use App\Application\Exception\AuthProviderException;
use App\Application\Exception\AuthProviderNotFound;
use Psr\Container\ContainerInterface;
use Spiral\Core\Attribute\Singleton;

#[Singleton]
final class AuthProviderService implements AuthProviderRegistryInterface
{
    /** @var array<non-empty-string, class-string<AuthProviderInterface>> */
    private array $providers = [];

    public function __construct(
        private readonly ContainerInterface $container,
    ) {}

    public function register(string $name, string $provider): void
    {
        if (!\is_subclass_of($provider, AuthProviderInterface::class)) {
            throw new AuthProviderException(
                \sprintf('Provider "%s" must implement AuthProviderInterface', $provider),
            );
        }

        $this->providers[$name] = $provider;
    }

    public function get(string $name): AuthProviderInterface
    {
        if (!isset($this->providers[$name])) {
            throw new AuthProviderNotFound(\sprintf('Auth provider "%s" not found', $name));
        }

        return $this->container->get($this->providers[$name]);
    }
}
