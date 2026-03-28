<?php

declare(strict_types=1);

namespace Tests\Unit\Modules\Sms\Domain;

use Modules\Sms\Domain\SmsMessage;
use PHPUnit\Framework\TestCase;

final class SmsMessageTest extends TestCase
{
    public function testConstructAndFields(): void
    {
        $sms = new SmsMessage(
            from: '+1234567890',
            to: '+0987654321',
            message: 'Hello world',
            gateway: 'twilio',
        );

        $this->assertSame('+1234567890', $sms->from);
        $this->assertSame('+0987654321', $sms->to);
        $this->assertSame('Hello world', $sms->message);
        $this->assertSame('twilio', $sms->gateway);
    }

    public function testJsonSerialize(): void
    {
        $sms = new SmsMessage(
            from: '+1234567890',
            to: '+0987654321',
            message: 'Hello world',
            gateway: 'twilio',
        );

        $this->assertSame([
            'from' => '+1234567890',
            'to' => '+0987654321',
            'message' => 'Hello world',
            'gateway' => 'twilio',
        ], $sms->jsonSerialize());
    }

    public function testJsonEncode(): void
    {
        $sms = new SmsMessage(
            from: '+1',
            to: '+2',
            message: 'Test',
            gateway: 'generic',
        );

        $json = \json_encode($sms, JSON_THROW_ON_ERROR);
        $decoded = \json_decode($json, true, 512, JSON_THROW_ON_ERROR);

        $this->assertSame('+1', $decoded['from']);
        $this->assertSame('+2', $decoded['to']);
        $this->assertSame('Test', $decoded['message']);
        $this->assertSame('generic', $decoded['gateway']);
    }
}
