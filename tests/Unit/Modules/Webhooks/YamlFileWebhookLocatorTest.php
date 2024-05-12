<?php

declare(strict_types=1);

namespace Tests\Unit\Modules\Webhooks;

use Modules\Webhooks\Application\Locator\Webhook;
use Modules\Webhooks\Application\Locator\WebhookFilesFinderInterface;
use Modules\Webhooks\Application\Locator\WebhookLocatorInterface;
use Tests\TestCase;

final class YamlFileWebhookLocatorTest extends TestCase
{
    public function testLocateFiles(): void
    {
        $finder = $this->mockContainer(WebhookFilesFinderInterface::class);

        $finder->shouldReceive('find')
            ->once()
            ->andReturn([
                $webhook1 = new Webhook(
                    key: 'file1',
                    event: 'event1',
                    url: 'url1',
                    headers: ['header1' => ['value1']],
                    verifySsl: true,
                    retryOnFailure: false,
                ),
                $webhook2 = new Webhook(
                    key: 'file2',
                    event: 'event2',
                    url: 'url2',
                    headers: ['header2' => ['value2']],
                    verifySsl: false,
                    retryOnFailure: true,
                ),
            ]);

        $this->assertSame(
            [$webhook1, $webhook2],
            \iterator_to_array($this->get(WebhookLocatorInterface::class)->findAll()),
        );
    }
}
