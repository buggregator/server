<?php

declare(strict_types=1);

namespace Interfaces\Http\Auth\SSO;

use App\Application\OAuth\AuthProviderInterface;
use App\Application\OAuth\User;
use Nyholm\Psr7\Uri;
use Tests\App\Http\ResponseAssertions;
use Tests\Feature\Interfaces\Http\ControllerTestCase;

final class LoginActionTest extends ControllerTestCase
{
    private \Mockery\MockInterface|AuthProviderInterface $auth;

    protected function setUp(): void
    {
        parent::setUp();

        $this->auth = $this->mockContainer(AuthProviderInterface::class);
    }

    public function testLoginUnauthenticated(): void
    {
        $this->auth->shouldReceive('isAuthenticated')
            ->once()
            ->andReturnFalse();

        $this->auth->shouldReceive('getLoginUrl')
            ->once()
            ->andReturn($uri = new Uri('https://example.com/login'));

        /** @var ResponseAssertions $response */
        $response = $this->http->get('/auth/sso/login');

        $response->assertRedirect(uri: (string) $uri);
    }

    public function testLoginAuthenticated(): void
    {
        $this->auth->shouldReceive('isAuthenticated')
            ->once()
            ->andReturnTrue();

        $this->auth->shouldReceive('getUser')
            ->once()
            ->andReturn(
                $user = new User(
                    provider: 'fake',
                    username: 'johndoe',
                    avatar: 'https://example.com/avatar.jpg',
                    email: 'johndoe@site.com',
                ),
            );

        /** @var ResponseAssertions $response */
        $response = $this->http->get('/auth/sso/login');

        $response->assertRedirect(uri: static fn(string $location) => \str_starts_with($location, '/#/login?token='));
    }
}
