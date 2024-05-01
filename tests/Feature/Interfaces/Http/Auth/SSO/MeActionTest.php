<?php

declare(strict_types=1);

namespace Interfaces\Http\Auth\SSO;

use App\Application\HTTP\Response\UserResource;
use App\Application\OAuth\ActorProvider;
use App\Application\OAuth\AuthProviderInterface;
use App\Application\OAuth\User;
use Nyholm\Psr7\Uri;
use Tests\App\Http\ResponseAssertions;
use Tests\Feature\Interfaces\Http\ControllerTestCase;

final class MeActionTest extends ControllerTestCase
{
    private \Mockery\MockInterface|AuthProviderInterface $auth;

    protected function setUp(): void
    {
        parent::setUp();

        $this->auth = $this->mockContainer(AuthProviderInterface::class);
    }

    public function testShowProfile(): void
    {
        $this->auth
            ->shouldReceive('getLogoutUrl')
            ->once()
            ->andReturn($uri = new Uri('https://example.com/logout'));

        $user = new User(
            provider: 'provider',
            username: 'user',
            avatar: 'https://example.com/avatar',
            email: 'user',
        );

        /** @var ResponseAssertions $response */
        $response = $this->http->authenticate($user)->get('/api/me');

        $response->assertResource(
            new UserResource($user, $uri),
        );
    }

    public function testGuestProfile(): void
    {
        $this->auth
            ->shouldReceive('getLogoutUrl')
            ->once()
            ->andReturn($uri = new Uri('https://example.com/logout'));

        /** @var ResponseAssertions $response */
        $response = $this->http->get('/api/me');

        $response->assertResource(
            new UserResource(ActorProvider::getGuestPayload(), $uri),
        );
    }
}
