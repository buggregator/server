<?php

declare(strict_types=1);

namespace App\Integration\Kinde;

use App\Application\Bootloader\AuthBootloader;
use App\Application\OAuth\AuthProviderRegistryInterface;
use Spiral\Boot\Bootloader\Bootloader;
use Kinde\KindeSDK\KindeClientSDK;
use Kinde\KindeSDK\Configuration;
use Kinde\KindeSDK\Sdk\Enums\GrantType;
use Spiral\Boot\EnvironmentInterface;
use Spiral\Session\SessionScope;

final class KindeBootloader extends Bootloader
{
    public function defineDependencies(): array
    {
        return [
            AuthBootloader::class,
        ];
    }

    public function defineSingletons(): array
    {
        return [
            SessionStorage::class => static fn(
                SessionScope $session,
            ): SessionStorage => new SessionStorage(
                section: $session->getSection('kinde'),
            ),
            AuthProvider::class => AuthProvider::class,
        ];
    }

    public function defineBindings(): array
    {
        return [
            Client::class => static function (
                EnvironmentInterface $env,
                Configuration $config,
                SessionStorage $storage,
            ): Client {
                return new Client(
                    domain: $config->getHost(),
                    redirectUri: $env->get('AUTH_CALLBACK_URL'),
                    clientId: $env->get('AUTH_CLIENT_ID'),
                    clientSecret: $env->get('AUTH_CLIENT_SECRET'),
                    logoutRedirectUri: $env->get('AUTH_LOGOUT_URL'),
                    storage: $storage,
                    client: new \GuzzleHttp\Client(),
                );
            },

            Configuration::class => static fn(
                EnvironmentInterface $env,
            ) => (new Configuration())->setHost($env->get('AUTH_PROVIDER_URL')),
        ];
    }

    public function init(AuthProviderRegistryInterface $provider): void
    {
        $provider->register('kinde', AuthProvider::class);
    }
}
