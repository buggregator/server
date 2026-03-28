<?php

declare(strict_types=1);

namespace Tests\Feature\Interfaces\Http\Sms;

use Nyholm\Psr7\Stream;
use Tests\Feature\Interfaces\Http\ControllerTestCase;

final class SmsHandlerTest extends ControllerTestCase
{
    public function testTwilioPayload(): void
    {
        $payload = \json_encode([
            'MessageSid' => 'SM1234567890',
            'From' => '+14155551234',
            'To' => '+14155555678',
            'Body' => 'Hello from Twilio',
        ]);

        $this->http->postJson(
            uri: '/sms',
            data: Stream::create($payload),
        )->assertOk();

        $this->broadcastig->assertPushed('events.project.default', function (array $data) {
            $this->assertSame('event.received', $data['event']);
            $this->assertSame('sms', $data['data']['type']);
            $this->assertSame('+14155551234', $data['data']['payload']['from']);
            $this->assertSame('+14155555678', $data['data']['payload']['to']);
            $this->assertSame('Hello from Twilio', $data['data']['payload']['message']);
            $this->assertSame('twilio', $data['data']['payload']['gateway']);

            return true;
        });
    }

    public function testVonagePayload(): void
    {
        $payload = \json_encode([
            'api_key' => 'test-key',
            'api_secret' => 'test-secret',
            'from' => '+31612345678',
            'to' => '+31698765432',
            'text' => 'Hello from Vonage',
        ]);

        $this->http->postJson(
            uri: '/sms',
            data: Stream::create($payload),
        )->assertOk();

        $this->broadcastig->assertPushed('events.project.default', function (array $data) {
            $this->assertSame('sms', $data['data']['type']);
            $this->assertSame('+31612345678', $data['data']['payload']['from']);
            $this->assertSame('+31698765432', $data['data']['payload']['to']);
            $this->assertSame('Hello from Vonage', $data['data']['payload']['message']);
            $this->assertSame('vonage', $data['data']['payload']['gateway']);

            return true;
        });
    }

    public function testPlivoPayload(): void
    {
        $payload = \json_encode([
            'MessageUUID' => 'uuid-abc-123',
            'From' => '+1234567890',
            'To' => '+0987654321',
            'Text' => 'Hello from Plivo',
        ]);

        $this->http->postJson(
            uri: '/sms',
            data: Stream::create($payload),
        )->assertOk();

        $this->broadcastig->assertPushed('events.project.default', function (array $data) {
            $this->assertSame('sms', $data['data']['type']);
            $this->assertSame('plivo', $data['data']['payload']['gateway']);
            $this->assertSame('Hello from Plivo', $data['data']['payload']['message']);

            return true;
        });
    }

    public function testGenericPayload(): void
    {
        $payload = \json_encode([
            'From' => '+1111',
            'To' => '+2222',
            'content' => 'Generic SMS',
        ]);

        $this->http->postJson(
            uri: '/sms',
            data: Stream::create($payload),
        )->assertOk();

        $this->broadcastig->assertPushed('events.project.default', function (array $data) {
            $this->assertSame('sms', $data['data']['type']);
            $this->assertSame('generic', $data['data']['payload']['gateway']);
            $this->assertSame('Generic SMS', $data['data']['payload']['message']);

            return true;
        });
    }

    public function testSmsWithProject(): void
    {
        $this->createProject('myapp');

        $payload = \json_encode([
            'from' => '+1111',
            'to' => '+2222',
            'message' => 'Project SMS',
        ]);

        $this->http->postJson(
            uri: '/sms/myapp',
            data: Stream::create($payload),
        )->assertOk();

        $this->broadcastig->assertPushed('events.project.myapp', function (array $data) {
            $this->assertSame('sms', $data['data']['type']);
            $this->assertSame('myapp', $data['data']['project']);

            return true;
        });
    }

    public function testEmptyBodyPassesThrough(): void
    {
        $this->http->postJson(
            uri: '/sms',
            data: Stream::create(''),
        );

        $this->broadcastig->assertNotPushed('events.project.default');
    }

    public function testMissingToFieldPassesThrough(): void
    {
        $payload = \json_encode([
            'from' => '+1111',
            'message' => 'No recipient',
        ]);

        $this->http->postJson(
            uri: '/sms',
            data: Stream::create($payload),
        );

        $this->broadcastig->assertNotPushed('events.project.default');
    }

    public function testNonSmsPathPassesThrough(): void
    {
        $this->http->postJson(
            uri: '/',
            data: ['from' => '+1', 'to' => '+2', 'message' => 'test'],
            headers: ['X-Buggregator-Event' => 'http-dump'],
        )->assertOk();

        $this->broadcastig->assertPushed('events.project.default', function (array $data) {
            $this->assertSame('http-dump', $data['data']['type']);

            return true;
        });
    }

    // === Explicit gateway via URL segment ===

    public function testExplicitGatewayTwilioValid(): void
    {
        $payload = \json_encode([
            'MessageSid' => 'SM123',
            'From' => '+1111',
            'To' => '+2222',
            'Body' => 'Hello via /sms/twilio',
        ]);

        $this->http->postJson(
            uri: '/sms/twilio',
            data: Stream::create($payload),
        )->assertOk();

        $this->broadcastig->assertPushed('events.project.default', function (array $data) {
            $this->assertSame('twilio', $data['data']['payload']['gateway']);
            $this->assertArrayNotHasKey('warnings', $data['data']['payload']);

            return true;
        });
    }

    public function testExplicitGatewayTwilioMissingFields(): void
    {
        $payload = \json_encode([
            'From' => '+1111',
            'To' => '+2222',
            // Missing MessageSid and Body — required for twilio
        ]);

        $this->http->postJson(
            uri: '/sms/twilio',
            data: Stream::create($payload),
        )->assertStatus(422);

        // Event is still stored despite 422 response
        $this->broadcastig->assertPushed('events.project.default', function (array $data) {
            $this->assertSame('sms', $data['data']['type']);
            $this->assertSame('twilio', $data['data']['payload']['gateway']);
            $this->assertArrayHasKey('warnings', $data['data']['payload']);
            $this->assertNotEmpty($data['data']['payload']['warnings']);

            return true;
        });
    }

    public function testExplicitGatewayWithProject(): void
    {
        $this->createProject('myapp');

        $payload = \json_encode([
            'MessageSid' => 'SM123',
            'From' => '+1111',
            'To' => '+2222',
            'Body' => 'With project',
        ]);

        $this->http->postJson(
            uri: '/sms/twilio/myapp',
            data: Stream::create($payload),
        )->assertOk();

        $this->broadcastig->assertPushed('events.project.myapp', function (array $data) {
            $this->assertSame('twilio', $data['data']['payload']['gateway']);
            $this->assertSame('myapp', $data['data']['project']);

            return true;
        });
    }

    public function testUnknownGatewayInUrlTreatedAsProject(): void
    {
        $this->createProject('myproject');

        $payload = \json_encode([
            'MessageSid' => 'SM123',
            'From' => '+1111',
            'To' => '+2222',
            'Body' => 'Auto-detected, segment is project',
        ]);

        // 'myproject' is not a known gateway, so treated as project name in auto-detect mode
        $this->http->postJson(
            uri: '/sms/myproject',
            data: Stream::create($payload),
        )->assertOk();

        $this->broadcastig->assertPushed('events.project.myproject', function (array $data) {
            $this->assertSame('twilio', $data['data']['payload']['gateway']);
            $this->assertSame('myproject', $data['data']['project']);

            return true;
        });
    }
}
