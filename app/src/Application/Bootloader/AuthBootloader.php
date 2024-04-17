<?php

declare(strict_types=1);

namespace App\Application\Bootloader;

use App\Application\Auth\AuthSettings;
use App\Application\Auth\JWTTokenStorage;
use App\Application\Auth\SuccessRedirect;
use App\Application\OAuth\ActorProvider;
use App\Application\OAuth\SessionStore;
use Psr\Http\Message\UriFactoryInterface;
use Spiral\Boot\Bootloader\Bootloader;

use Auth0\SDK\Auth0;
use Auth0\SDK\Configuration\SdkConfiguration;
use Spiral\Boot\EnvironmentInterface;
use Spiral\Bootloader\Auth\HttpAuthBootloader;
use Spiral\Core\Container\Autowire;
use Spiral\Http\ResponseWrapper;
use Spiral\Session\SessionScope;

final class AuthBootloader extends Bootloader
{
    public function defineBindings(): array
    {
        return [
            Auth0::class => static fn(SdkConfiguration $config, SessionScope $session) => new Auth0(
                $config->setTransientStorage(new SessionStore($session)),
            ),

            SdkConfiguration::class => static fn(EnvironmentInterface $env) => new SdkConfiguration(
                strategy: $env->get('AUTH_STRATEGY', SdkConfiguration::STRATEGY_REGULAR),
                domain: $env->get('AUTH_PROVIDER_URL'),
                clientId: $env->get('AUTH_CLIENT_ID'),
                redirectUri: $env->get('AUTH_CALLBACK_URL'),
                clientSecret: $env->get('AUTH_CLIENT_SECRET'),
                scope: \explode(',', $env->get('AUTH_SCOPES', 'openid,profile,email')),
                cookieSecret: $env->get('AUTH_COOKIE_SECRET', $env->get('ENCRYPTER_KEY') ?? 'secret'),
            ),
        ];
    }

    public function defineSingletons(): array
    {
        return [
            AuthSettings::class => static fn(
                EnvironmentInterface $env,
                UriFactoryInterface $factory,
            ) => new AuthSettings(
                enabled: $env->get('AUTH_ENABLED', false),
                loginUrl: $factory->createUri('/auth/sso/login'),
            ),

            SuccessRedirect::class => static fn(
                UriFactoryInterface $factory,
                ResponseWrapper $response,
            ) => new SuccessRedirect(
                response: $response,
                redirectUrl: $factory->createUri('/#/login'),
            ),
        ];
    }

    public function init(
        HttpAuthBootloader $httpAuth,
        EnvironmentInterface $env,
        \Spiral\Bootloader\Auth\AuthBootloader $auth,
    ): void {
        $auth->addActorProvider(new Autowire(ActorProvider::class));
        $httpAuth->addTokenStorage(
            'jwt',
            new Autowire(
                JWTTokenStorage::class,
                [
                    'secret' => $env->get('AUTH_JWT_SECRET', $env->get('ENCRYPTER_KEY')),
                ],
            ),
        );
    }
}
