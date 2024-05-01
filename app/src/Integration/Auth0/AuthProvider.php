<?php

declare(strict_types=1);

namespace App\Integration\Auth0;

use App\Application\OAuth\AuthProviderInterface;
use App\Application\OAuth\User;
use Auth0\SDK\Auth0;
use Nyholm\Psr7\Uri;
use Psr\Container\ContainerInterface;
use Psr\Http\Message\UriInterface;
use Spiral\Core\Attribute\Proxy;
use Spiral\Http\Request\InputManager;

final readonly class AuthProvider implements AuthProviderInterface
{
    public function __construct(
        #[Proxy]
        private ContainerInterface $container,
    ) {}

    public function getLoginUrl(): UriInterface
    {
        $auth = $this->getProvider();

        if ($this->isAuthenticated()) {
            throw new \RuntimeException('User is already authenticated');
        }

        return new Uri($auth->login());
    }

    public function isAuthenticated(): bool
    {
        $auth = $this->getProvider();
        $session = $auth->getCredentials();

        return null !== $session && !$session->accessTokenExpired;
    }

    public function getUser(): ?User
    {
        $payload = $this->getProvider()->getUser();

        if (null === $payload) {
            return null;
        }

        return new Auth0User($payload);
    }

    public function authenticate(InputManager $input): void
    {
        $this->getProvider()->exchange(
            code: $input->query('code'),
            state: $input->query('state'),
        );
    }

    private function getProvider(): Auth0
    {
        return $this->container->get(Auth0::class);
    }

    public function getLogoutUrl(): ?UriInterface
    {
        return new Uri($this->getProvider()->authentication()->getLogoutLink());
    }

    public function logout(): void
    {
        $this->getProvider()->logout();
    }
}
