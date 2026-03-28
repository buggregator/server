<?php

declare(strict_types=1);

namespace Tests\Unit\Modules\Sms\Application\Gateway;

use Modules\Sms\Application\Gateway\FieldMapGateway;
use Modules\Sms\Application\Gateway\GatewayRegistry;
use PHPUnit\Framework\TestCase;

final class GatewayRegistryTest extends TestCase
{
    public function testDetectsTwilioPayload(): void
    {
        $registry = $this->makeRegistry();

        $gateway = $registry->detect([
            'MessageSid' => 'SM123',
            'Body' => 'Hello',
            'From' => '+1',
            'To' => '+2',
        ]);

        $this->assertNotNull($gateway);
        $this->assertSame('twilio', $gateway->name());
    }

    public function testDetectsVonagePayload(): void
    {
        $registry = $this->makeRegistry();

        $gateway = $registry->detect([
            'messageId' => 'msg-123',
            'text' => 'Hello',
            'msisdn' => '+1',
            'to' => '+2',
        ]);

        $this->assertNotNull($gateway);
        $this->assertSame('vonage', $gateway->name());
    }

    public function testDetectsPlivoPayload(): void
    {
        $registry = $this->makeRegistry();

        $gateway = $registry->detect([
            'MessageUUID' => 'uuid-123',
            'Text' => 'Hello',
            'From' => '+1',
            'To' => '+2',
        ]);

        $this->assertNotNull($gateway);
        $this->assertSame('plivo', $gateway->name());
    }

    public function testFallsBackToGenericForUnknownPayload(): void
    {
        $registry = $this->makeRegistry();

        $gateway = $registry->detect([
            'from' => '+1',
            'to' => '+2',
            'message' => 'Hello',
        ]);

        $this->assertNotNull($gateway);
        $this->assertSame('generic', $gateway->name());
    }

    public function testReturnsNullWhenNoGatewaysRegistered(): void
    {
        $registry = new GatewayRegistry([]);

        $this->assertNull($registry->detect(['from' => '+1']));
    }

    public function testPriorityOrderFirstMatchWins(): void
    {
        // Both gateways would match, but first one wins
        $first = new FieldMapGateway('first', [], ['from'], ['to'], ['message']);
        $second = new FieldMapGateway('second', [], ['from'], ['to'], ['message']);

        $registry = new GatewayRegistry([$first, $second]);

        $gateway = $registry->detect(['from' => '+1', 'to' => '+2', 'message' => 'Hi']);
        $this->assertSame('first', $gateway->name());
    }

    private function makeRegistry(): GatewayRegistry
    {
        return new GatewayRegistry([
            new FieldMapGateway(
                gatewayName: 'twilio',
                detectFields: ['MessageSid', 'Body'],
                fromFields: ['From'],
                toFields: ['To'],
                messageFields: ['Body'],
            ),
            new FieldMapGateway(
                gatewayName: 'vonage',
                detectFields: ['messageId', 'text'],
                fromFields: ['msisdn', 'from'],
                toFields: ['to'],
                messageFields: ['text'],
            ),
            new FieldMapGateway(
                gatewayName: 'plivo',
                detectFields: ['MessageUUID', 'Text'],
                fromFields: ['From', 'src'],
                toFields: ['To', 'dst'],
                messageFields: ['Text', 'text'],
            ),
            new FieldMapGateway(
                gatewayName: 'generic',
                detectFields: [],
                fromFields: ['from', 'From', 'sender', 'msisdn'],
                toFields: ['to', 'To', 'recipient', 'dst'],
                messageFields: ['message', 'body', 'Body', 'text', 'Text', 'content'],
            ),
        ]);
    }
}
