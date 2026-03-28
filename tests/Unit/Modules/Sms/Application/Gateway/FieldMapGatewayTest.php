<?php

declare(strict_types=1);

namespace Tests\Unit\Modules\Sms\Application\Gateway;

use Modules\Sms\Application\Gateway\FieldMapGateway;
use PHPUnit\Framework\TestCase;

final class FieldMapGatewayTest extends TestCase
{
    public function testTwilioDetectsWhenSignatureFieldsPresent(): void
    {
        $gateway = $this->makeTwilioGateway();

        $this->assertTrue($gateway->detect([
            'MessageSid' => 'SM123',
            'Body' => 'Hello',
            'From' => '+1234',
            'To' => '+5678',
        ]));
    }

    public function testTwilioDoesNotDetectWhenFieldsMissing(): void
    {
        $gateway = $this->makeTwilioGateway();

        $this->assertFalse($gateway->detect([
            'From' => '+1234',
            'To' => '+5678',
            'Body' => 'Hello',
        ]));
    }

    public function testTwilioParsesPayload(): void
    {
        $gateway = $this->makeTwilioGateway();

        $sms = $gateway->parse([
            'MessageSid' => 'SM123',
            'From' => '+1234567890',
            'To' => '+0987654321',
            'Body' => 'Hello from Twilio',
        ]);

        $this->assertSame('+1234567890', $sms->from);
        $this->assertSame('+0987654321', $sms->to);
        $this->assertSame('Hello from Twilio', $sms->message);
        $this->assertSame('twilio', $sms->gateway);
    }

    public function testVonageDetectsWhenSignatureFieldsPresent(): void
    {
        $gateway = $this->makeVonageGateway();

        $this->assertTrue($gateway->detect([
            'messageId' => 'msg-123',
            'text' => 'Hello',
            'msisdn' => '+1234',
            'to' => '+5678',
        ]));
    }

    public function testVonageDoesNotDetectWithoutSignatureFields(): void
    {
        $gateway = $this->makeVonageGateway();

        $this->assertFalse($gateway->detect([
            'from' => '+1234',
            'to' => '+5678',
            'text' => 'Hello',
        ]));
    }

    public function testVonageParsesPayload(): void
    {
        $gateway = $this->makeVonageGateway();

        $sms = $gateway->parse([
            'messageId' => 'msg-123',
            'msisdn' => '+1234567890',
            'to' => '+0987654321',
            'text' => 'Hello from Vonage',
        ]);

        $this->assertSame('+1234567890', $sms->from);
        $this->assertSame('+0987654321', $sms->to);
        $this->assertSame('Hello from Vonage', $sms->message);
        $this->assertSame('vonage', $sms->gateway);
    }

    public function testPlivoDetectsAndParses(): void
    {
        $gateway = $this->makePlivoGateway();

        $body = [
            'MessageUUID' => 'uuid-123',
            'From' => '+1234567890',
            'To' => '+0987654321',
            'Text' => 'Hello from Plivo',
        ];

        $this->assertTrue($gateway->detect($body));

        $sms = $gateway->parse($body);
        $this->assertSame('+1234567890', $sms->from);
        $this->assertSame('+0987654321', $sms->to);
        $this->assertSame('Hello from Plivo', $sms->message);
        $this->assertSame('plivo', $sms->gateway);
    }

    public function testPlivoParsesSrcDstFields(): void
    {
        $gateway = $this->makePlivoGateway();

        $sms = $gateway->parse([
            'MessageUUID' => 'uuid-123',
            'src' => '+111',
            'dst' => '+222',
            'Text' => 'Alt fields',
        ]);

        $this->assertSame('+111', $sms->from);
        $this->assertSame('+222', $sms->to);
    }

    public function testGenericAlwaysDetects(): void
    {
        $gateway = $this->makeGenericGateway();

        $this->assertTrue($gateway->detect([]));
        $this->assertTrue($gateway->detect(['anything' => 'here']));
    }

    public function testGenericParsesCommonFieldNames(): void
    {
        $gateway = $this->makeGenericGateway();

        $sms = $gateway->parse([
            'from' => '+1234',
            'to' => '+5678',
            'message' => 'Generic message',
        ]);

        $this->assertSame('+1234', $sms->from);
        $this->assertSame('+5678', $sms->to);
        $this->assertSame('Generic message', $sms->message);
        $this->assertSame('generic', $sms->gateway);
    }

    public function testGenericTriesAlternativeFieldNames(): void
    {
        $gateway = $this->makeGenericGateway();

        $sms = $gateway->parse([
            'sender' => '+1234',
            'recipient' => '+5678',
            'body' => 'Alt generic',
        ]);

        $this->assertSame('+1234', $sms->from);
        $this->assertSame('+5678', $sms->to);
        $this->assertSame('Alt generic', $sms->message);
    }

    public function testReturnsEmptyStringForMissingFields(): void
    {
        $gateway = $this->makeGenericGateway();

        $sms = $gateway->parse([]);

        $this->assertSame('', $sms->from);
        $this->assertSame('', $sms->to);
        $this->assertSame('', $sms->message);
    }

    public function testName(): void
    {
        $this->assertSame('twilio', $this->makeTwilioGateway()->name());
        $this->assertSame('vonage', $this->makeVonageGateway()->name());
        $this->assertSame('plivo', $this->makePlivoGateway()->name());
        $this->assertSame('generic', $this->makeGenericGateway()->name());
    }

    private function makeTwilioGateway(): FieldMapGateway
    {
        return new FieldMapGateway(
            gatewayName: 'twilio',
            detectFields: ['MessageSid', 'Body'],
            fromFields: ['From'],
            toFields: ['To'],
            messageFields: ['Body'],
        );
    }

    private function makeVonageGateway(): FieldMapGateway
    {
        return new FieldMapGateway(
            gatewayName: 'vonage',
            detectFields: ['messageId', 'text'],
            fromFields: ['msisdn', 'from'],
            toFields: ['to'],
            messageFields: ['text'],
        );
    }

    private function makePlivoGateway(): FieldMapGateway
    {
        return new FieldMapGateway(
            gatewayName: 'plivo',
            detectFields: ['MessageUUID', 'Text'],
            fromFields: ['From', 'src'],
            toFields: ['To', 'dst'],
            messageFields: ['Text', 'text'],
        );
    }

    private function makeGenericGateway(): FieldMapGateway
    {
        return new FieldMapGateway(
            gatewayName: 'generic',
            detectFields: [],
            fromFields: ['from', 'From', 'sender', 'msisdn'],
            toFields: ['to', 'To', 'recipient', 'dst'],
            messageFields: ['message', 'body', 'Body', 'text', 'Text', 'content'],
        );
    }
}
