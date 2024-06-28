<?php

declare(strict_types=1);

namespace Tests\Feature\Interfaces\Jobs\Webhooks;

use Database\Factory\WebhookFactory;
use GuzzleHttp\ClientInterface;
use Mockery\MockInterface;
use Modules\Webhooks\Domain\Delivery;
use Modules\Webhooks\Domain\Webhook;
use Modules\Webhooks\Domain\WebhookEvent;
use Modules\Webhooks\Domain\WebhookServiceInterface;
use Modules\Webhooks\Interfaces\Job\RetryPolicy;
use Nyholm\Psr7\Stream;
use Psr\Http\Message\ResponseInterface;
use Spiral\DatabaseSeeder\Database\EntityAssertion;
use Tests\DatabaseTestCase;

final class WebhookHandlerTest extends DatabaseTestCase
{
    private MockInterface|ClientInterface $httpClient;

    private EntityAssertion $assertion;

    private WebhookServiceInterface $service;

    protected function setUp(): void
    {
        parent::setUp();

        $this->httpClient = $this->mockContainer(ClientInterface::class);
        $this->assertion = $this->assertEntity(Delivery::class);
        $this->service = $this->get(WebhookServiceInterface::class);
    }

    public function testRetryPolicyWithoutRetries(): void
    {
        /** @var Webhook $webhook */
        $webhook = WebhookFactory::new()->withRetry()->createOne();
        $retryPolicy = $this->mockContainer(RetryPolicy::class);

        $retryPolicy->shouldReceive('canRetry')
            ->once()
            ->andReturnFalse();

        $this->service->send(
            new WebhookEvent(
                event: $webhook->event,
                payload: $payload = ['foo' => 'bar'],
            ),
        );
    }

    public function testRetryPolicyWithRetries(): void
    {
        /** @var Webhook $webhook */
        $webhook = WebhookFactory::new()->withRetry()->createOne();
        $retryPolicy = $this->mockContainer(RetryPolicy::class);

        $retryPolicy->shouldReceive('canRetry')
            ->once()
            ->andReturnTrue();

        $retryPolicy->shouldReceive('nextRetry')->once();

        $retryPolicy->shouldReceive('canRetry')
            ->once()
            ->andReturnFalse();

        $this->httpClient->shouldReceive('request')
            ->once()
            ->andThrow(new \Exception('Something went wrong'));

        $this->service->send(
            new WebhookEvent(
                event: $webhook->event,
                payload: $payload = ['foo' => 'bar'],
            ),
        );
    }

    public function testRetryPolicyWithMaxRetries(): void
    {
        /** @var Webhook $webhook */
        $webhook = WebhookFactory::new()->withoutRetry()->createOne();
        $retryPolicy = $this->mockContainer(RetryPolicy::class);

        $retryPolicy->shouldReceive('setMaxRetries')
            ->once()
            ->with(1)
            ->andReturnSelf();

        $retryPolicy->shouldReceive('canRetry')
            ->once()
            ->andReturnFalse();

        $this->service->send(
            new WebhookEvent(
                event: $webhook->event,
                payload: $payload = ['foo' => 'bar'],
            ),
        );
    }

    public function testSendMultipleWebhooks(): void
    {
        $sentWebhooks = WebhookFactory::new()
            ->forEvent($event = 'sentry.received')
            ->withRetry()
            ->times(2)
            ->create();

        $missedWebhooks = WebhookFactory::new()
            ->forEvent('monolog.received')
            ->withRetry()
            ->create();

        $retryPolicy = $this->mockContainer(RetryPolicy::class);

        $retryPolicy->shouldReceive('canRetry')
            ->twice()
            ->andReturnTrue();

        $this->httpClient->shouldReceive('request')
            ->twice()
            ->andReturn($response = \Mockery::mock(ResponseInterface::class));

        $response->shouldReceive('getBody')
            ->twice()
            ->andReturnUsing(static fn() => Stream::create('{"status":"ok"}'));

        $response->shouldReceive('getStatusCode')
            ->twice()
            ->andReturn(200);

        $this->service->send(
            new WebhookEvent(
                event: $event,
                payload: $payload = ['foo' => 'bar'],
            ),
        );

        foreach ($sentWebhooks as $webhook) {
            $this->assertion
                ->where([
                    'webhook_uuid' => (string) $webhook->getUuid(),
                    'payload' => '{"foo":"bar"}',
                    'status' => 200,
                    'response' => '{"status":"ok"}',
                ])
                ->assertExists();
        }

        foreach ($missedWebhooks as $webhook) {
            $this->assertion
                ->where([
                    'webhook_uuid' => (string) $webhook->getUuid(),
                ])
                ->assertMissing();
        }
    }
}
