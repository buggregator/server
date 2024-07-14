<?php

declare(strict_types=1);

namespace App\Integration\Kinde;

use App\Application\OAuth\AuthProviderInterface;
use App\Application\OAuth\User;
use Kinde\KindeSDK\Configuration;
use Kinde\KindeSDK\Sdk\Utils\Utils;
use Psr\Container\ContainerInterface;
use Psr\Http\Message\UriInterface;
use Spiral\Core\Attribute\Proxy;
use Spiral\Http\Request\InputManager;

final readonly class AuthProvider implements AuthProviderInterface
{
    public function __construct(
        #[Proxy]
        private ContainerInterface $container,
        private Configuration $config,
        private SessionStorage $storage,
    ) {}

    public function getLoginUrl(): UriInterface
    {
        $state = Utils::randomString();
        $this->storage->setState($state);

        return $this->getProvider()->getLoginUrl();
    }

    public function isAuthenticated(): bool
    {
        return $this->getProvider()->isAuthenticated();
    }

    public function getUser(): ?User
    {
        $payload = $this->getProvider()->getUserDetails();

        if ($payload === []) {
            return null;
        }

        return new KindeUser($payload);
    }

    public function authenticate(InputManager $input): void
    {
        $token = $this->getProvider()->getToken([
            'code' => $input->query('code'),
            'state' => $input->query('state'),
        ]);

        $this->config->setAccessToken($token->access_token);
    }

    private function getProvider(): Client
    {
        return $this->container->get(Client::class);
    }

    public function getLogoutUrl(): ?UriInterface
    {
        return $this->getProvider()->getLogoutUrl();
    }

    public function logout(): void
    {
        $this->getProvider()->cleanStorage();
    }
}
