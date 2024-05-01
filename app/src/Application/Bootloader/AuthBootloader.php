<?php

declare(strict_types=1);

namespace App\Application\Bootloader;

use App\Application\Auth\AuthSettings;
use App\Application\Auth\JWTTokenStorage;
use App\Application\Auth\SuccessRedirect;
use App\Application\OAuth\ActorProvider;
use App\Application\OAuth\AuthProviderInterface;
use App\Application\OAuth\AuthProviderRegistryInterface;
use App\Application\OAuth\AuthProviderService;
use Psr\Http\Message\UriFactoryInterface;
use Spiral\Boot\Bootloader\Bootloader;
use Spiral\Boot\EnvironmentInterface;
use Spiral\Bootloader\Auth\HttpAuthBootloader;
use Spiral\Core\Container\Autowire;
use Spiral\Http\ResponseWrapper;

final class AuthBootloader extends Bootloader
{
    public function defineSingletons(): array
    {
        return [
            AuthProviderInterface::class => static fn(
                EnvironmentInterface $env,
                AuthProviderService $service,
            ) => $service->get(
                name: $env->get('AUTH_PROVIDER', 'auth0'),
            ),

            AuthProviderService::class => AuthProviderService::class,
            AuthProviderRegistryInterface::class => AuthProviderService::class,

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
