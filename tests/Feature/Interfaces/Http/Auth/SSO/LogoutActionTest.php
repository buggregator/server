<?php

declare(strict_types=1);

namespace Interfaces\Http\Auth\SSO;

use Mockery\MockInterface;
use App\Application\OAuth\AuthProviderInterface;
use Tests\Feature\Interfaces\Http\ControllerTestCase;

final class LogoutActionTest extends ControllerTestCase
{
    private MockInterface|AuthProviderInterface $auth;

    protected function setUp(): void
    {
        parent::setUp();

        $this->auth = $this->mockContainer(AuthProviderInterface::class);
    }

    public function testLogout(): void
    {
        $this->auth
            ->shouldReceive('logout')
            ->once();

        $response = $this->http->get('/auth/sso/logout');

        $response->assertStatus(200);
    }
}
