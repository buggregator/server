<?php

declare(strict_types=1);

namespace Interfaces\Http\Auth\SSO;

use Mockery\MockInterface;
use Spiral\Http\Request\InputManager;
use App\Application\OAuth\AuthProviderInterface;
use App\Application\OAuth\User;
use Tests\App\Http\ResponseAssertions;
use Tests\Feature\Interfaces\Http\ControllerTestCase;

final class CallbackActionTest extends ControllerTestCase
{
    private MockInterface|AuthProviderInterface $auth;

    protected function setUp(): void
    {
        parent::setUp();

        $this->auth = $this->mockContainer(AuthProviderInterface::class);
    }

    public function testCallback(): void
    {
        $this->auth
            ->shouldReceive('authenticate')
            ->once()
            ->with(\Mockery::type(InputManager::class));

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
        $response = $this->http->get('/auth/sso/callback', [
            'code' => 'code_123',
            'state' => 'state_123',
        ]);

        $response->assertRedirect(uri: static fn(string $location) => \str_starts_with($location, '/#/login?token='));
    }
}
