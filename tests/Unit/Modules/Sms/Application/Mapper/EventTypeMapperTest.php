<?php

declare(strict_types=1);

namespace Tests\Unit\Modules\Sms\Application\Mapper;

use Modules\Sms\Application\Mapper\EventTypeMapper;
use Modules\Sms\Domain\SmsMessage;
use PHPUnit\Framework\TestCase;

final class EventTypeMapperTest extends TestCase
{
    private EventTypeMapper $mapper;

    protected function setUp(): void
    {
        $this->mapper = new EventTypeMapper();
    }

    public function testToPreviewFromArray(): void
    {
        $preview = $this->mapper->toPreview('sms', [
            'from' => '+1234',
            'to' => '+5678',
            'message' => 'Hello',
            'gateway' => 'twilio',
        ]);

        $this->assertSame([
            'from' => '+1234',
            'to' => '+5678',
            'message' => 'Hello',
            'gateway' => 'twilio',
        ], $preview);
    }

    public function testToPreviewFromJsonSerializable(): void
    {
        $sms = new SmsMessage(
            from: '+1234',
            to: '+5678',
            message: 'Hello',
            gateway: 'vonage',
        );

        $preview = $this->mapper->toPreview('sms', $sms);

        $this->assertSame([
            'from' => '+1234',
            'to' => '+5678',
            'message' => 'Hello',
            'gateway' => 'vonage',
        ], $preview);
    }

    public function testToPreviewWithMissingFields(): void
    {
        $preview = $this->mapper->toPreview('sms', []);

        $this->assertSame([
            'from' => '',
            'to' => '',
            'message' => '',
            'gateway' => '',
        ], $preview);
    }

    public function testToSearchableText(): void
    {
        $text = $this->mapper->toSearchableText('sms', [
            'from' => '+1234',
            'to' => '+5678',
            'message' => 'Hello world',
        ]);

        $this->assertSame('+1234 +5678 Hello world', $text);
    }

    public function testToSearchableTextFromJsonSerializable(): void
    {
        $sms = new SmsMessage(
            from: '+1234',
            to: '+5678',
            message: 'Test message',
            gateway: 'generic',
        );

        $text = $this->mapper->toSearchableText('sms', $sms);

        $this->assertSame('+1234 +5678 Test message', $text);
    }

    public function testToSearchableTextSkipsEmptyFields(): void
    {
        $text = $this->mapper->toSearchableText('sms', [
            'from' => '',
            'to' => '+5678',
            'message' => 'Hello',
        ]);

        $this->assertSame('+5678 Hello', $text);
    }
}
