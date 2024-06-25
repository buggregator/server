<?php

declare(strict_types=1);

namespace Tests\App\Http;

use Spiral\Auth\TokenInterface;
use App\Application\Domain\ValueObjects\Uuid;
use App\Application\OAuth\User;
use Spiral\Auth\TokenStorageInterface;
use Spiral\Auth\TokenStorageProviderInterface;
use Spiral\Testing\Http\FakeHttp;
use Spiral\Testing\Http\TestResponse;
use Tests\TestCase;

/**
 * @mixin FakeHttp
 */
final class HttpFaker
{
    private bool $dumpResponse = false;

    public function __construct(
        private FakeHttp $http,
        private TestCase $tests,
    ) {}

    public function authenticate(
        ?User $user = null,
    ): self {
        $user ??= new User(
            provider: 'fake',
            username: 'johndoe',
            avatar: 'https://example.com/avatar',
            email: 'user',
        );

        $hash = \Ramsey\Uuid\Uuid::uuid7()->toString();

        $tokenStorageProvider = $this->tests->mockContainer(TokenStorageProviderInterface::class);
        $tokenStorageProvider->shouldReceive('getStorage')->once()->andReturn(
            $tokenStorage = \Mockery::mock(TokenStorageInterface::class),
        );
        $tokenStorage->shouldReceive('load')->once()->with($hash)->andReturn(
            $token = \Mockery::mock(TokenInterface::class),
        );
        $token->shouldReceive('getID')->andReturn($hash);
        $token->shouldReceive('getExpiresAt')->andReturnNull();
        $token->shouldReceive('getPayload')->andReturn($user->jsonSerialize());

        $self = clone $this;
        $self->http = $this->http->withHeader('x-auth-token', $hash);

        return $self;
    }

    public function withWrongToken(string $token = 'auth-token'): self
    {
        $self = clone $this;
        $self->http = $this->http->withHeader('x-auth-token', $token);

        return $self;
    }

    public function listWebhooks(): ResponseAssertions
    {
        return $this->makeResponse(
            $this->http->getJson(uri: '/api/webhooks'),
        );
    }

    public function listWebhookDeliveries(Uuid $webhookUuid): ResponseAssertions
    {
        return $this->makeResponse(
            $this->http->getJson(uri: '/api/webhook/' . $webhookUuid . '/deliveries'),
        );
    }

    public function showEvent(Uuid $uuid): ResponseAssertions
    {
        return $this->makeResponse(
            $this->http->getJson(uri: '/api/event/' . $uuid),
        );
    }

    public function deleteEvent(Uuid $uuid): ResponseAssertions
    {
        return $this->makeResponse(
            $this->http->deleteJson(uri: '/api/event/' . $uuid),
        );
    }

    public function clearEvents(?string $type = null, ?string $project = null, ?array $uuids = null): ResponseAssertions
    {
        $args = [];
        if ($type) {
            $args['type'] = $type;
        }

        if ($project) {
            $args['project'] = $project;
        }

        if ($uuids) {
            $args['uuids'] = $uuids;
        }

        return $this->makeResponse(
            $this->http->deleteJson(
                uri: '/api/events/',
                data: $args,
            ),
        );
    }

    public function __call(string $name, array $arguments): ResponseAssertions|self
    {
        if (!method_exists($this->http, $name)) {
            throw new \Exception("Method $name does not exist");
        }

        if (\str_starts_with($name, 'with')) {
            $this->http->$name(...$arguments);
            return $this;
        }

        if ($name === 'getCookies') {
            return $this->getCookies();
        }

        return $this->makeResponse(
            $this->http->$name(...$arguments),
        );
    }

    private function makeResponse(TestResponse $response): ResponseAssertions
    {
        if ($this->dumpResponse) {
            $body = (string) $response;

            try {
                $body = \json_decode($body, true, 512, JSON_THROW_ON_ERROR);
            } catch (\Throwable) {
            }
        }

        return new ResponseAssertions($response);
    }
}
