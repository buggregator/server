<?php

declare(strict_types=1);

namespace App\Integration\Auth0;

use App\Application\Bootloader\AuthBootloader;
use App\Application\OAuth\AuthProviderRegistryInterface;
use Auth0\SDK\Auth0;
use Auth0\SDK\Configuration\SdkConfiguration;
use Spiral\Boot\Bootloader\Bootloader;
use Spiral\Boot\EnvironmentInterface;
use Spiral\Session\SessionScope;

final class Auth0Bootloader extends Bootloader
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
            AuthProvider::class => AuthProvider::class,
        ];
    }

    public function defineBindings(): array
    {
        return [
            Auth0::class => static fn(SdkConfiguration $config, SessionScope $session) => new Auth0(
                $config->setTransientStorage(new SessionStore($session->getSection('auth0'))),
            ),

            SdkConfiguration::class => static fn(EnvironmentInterface $env) => new SdkConfiguration(
                strategy: $env->get('AUTH_STRATEGY', SdkConfiguration::STRATEGY_REGULAR),
                domain: $env->get('AUTH_PROVIDER_URL'),
                clientId: $env->get('AUTH_CLIENT_ID'),
                redirectUri: $env->get('AUTH_CALLBACK_URL'),
                clientSecret: $env->get('AUTH_CLIENT_SECRET'),
                scope: \explode(',', (string) $env->get('AUTH_SCOPES', 'openid,profile,email')),
                cookieSecret: $env->get('AUTH_COOKIE_SECRET', $env->get('ENCRYPTER_KEY') ?? 'secret'),
            ),
        ];
    }

    public function init(AuthProviderRegistryInterface $provider): void
    {
        $provider->register('auth0', AuthProvider::class);
    }
}
