<?php

declare(strict_types=1);

namespace Tests\Feature\Interfaces\Http\Sentry;

use App\Application\Broadcasting\Channel\EventsChannel;
use Modules\Projects\Domain\Project;
use Modules\Projects\Domain\ValueObject\Key;
use Nyholm\Psr7\Stream;
use Tests\App\Http\ResponseAssertions;

final class SentryEventActionTest extends SentryControllerTestCase
{
    private Project $project;

    protected function setUp(): void
    {
        parent::setUp();

        $this->project = $this->createProject('default');
    }

    public function testSendWithoutGzip(): void
    {
        $payload = $this->makeEventPayload('Hello world');
        $this->makeRequest(payload: $payload, project: $this->project->getKey())->assertOk();

        $this->broadcastig->assertPushed(new EventsChannel($this->project->getKey()), function (array $data) {
            $this->assertSame('event.received', $data['event']);
            $this->assertSame('sentry', $data['data']['type']);
            $this->assertSame('default', $data['data']['project']);

            $this->assertSame('production', $data['data']['payload']['environment']);
            $this->assertSame('Hello world', $data['data']['payload']['message']);
            $this->assertSame('info', $data['data']['payload']['level']);

            $this->assertNotEmpty($data['data']['uuid']);
            $this->assertNotEmpty($data['data']['timestamp']);

            return true;
        });
    }

    private function makeRequest(
        string $payload,
        string $secret = 'secret',
        string|Key $project = 'default',
    ): ResponseAssertions {
        return $this->http
            ->postJson(
                uri: '/api/' . $project . '/envelope/',
                data: Stream::create($payload),
                headers: [
                    'X-Buggregator-Event' => 'sentry',
                    'Content-Type' => 'application/x-sentry-envelope',
                    'X-Sentry-Auth' => 'Sentry sentry_version=7, sentry_client=sentry.php/4.0.1, sentry_key=' . $secret,
                ],
            );
    }
}
