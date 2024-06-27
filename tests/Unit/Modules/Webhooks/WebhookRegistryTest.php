<?php

declare(strict_types=1);

namespace Tests\Unit\Modules\Webhooks;

use Modules\Webhooks\Application\Locator\Webhook;
use Modules\Webhooks\Application\Locator\WebhookRegistryInterface;
use Modules\Webhooks\Domain\Webhook as WebhookEntity;
use Modules\Webhooks\Exceptions\WebhooksAlreadyExistsException;
use Tests\DatabaseTestCase;

final class WebhookRegistryTest extends DatabaseTestCase
{
    private $registry;

    private $assert;

    protected function setUp(): void
    {
        parent::setUp();

        $this->registry = $this->get(WebhookRegistryInterface::class);
        $this->assert = $this->assertEntity(WebhookEntity::class);
    }

    public function testRegister(): void
    {
        $this->assert->assertCount(0);

        $this->registry->register($this->createWebhook());

        $this->assert->where([
            'key' => 'report-error',
            'event' => 'event',
            'url' => 'http://example.com',
            'verify_ssl' => true,
            'retry_on_failure' => true,
        ])->assertCount(1);
    }

    public function testRegisterNonUniqueKey(): void
    {
        $this->expectException(WebhooksAlreadyExistsException::class);
        $this->expectExceptionMessage('Webhook with key report-error already exists');

        $this->registry->register($this->createWebhook());
        $this->registry->register($this->createWebhook());
    }

    private function createWebhook(): Webhook
    {
        return new Webhook(
            key: 'report-error',
            event: 'event',
            url: 'http://example.com',
            headers: ['header-key' => ['value']],
            verifySsl: true,
            retryOnFailure: true,
        );
    }
}
