<?php

declare(strict_types=1);

namespace Tests\Unit\Modules\Smtp\Application\Mail\Strategy;

use Modules\Smtp\Application\Mail\Strategy\InlineAttachmentStrategy;
use Tests\TestCase;
use ZBateson\MailMimeParser\Message\IMessagePart;

final class InlineAttachmentStrategyTest extends TestCase
{
    private InlineAttachmentStrategy $strategy;

    protected function setUp(): void
    {
        parent::setUp();
        $this->strategy = new InlineAttachmentStrategy();
    }

    public function testCanHandleInlineAttachment(): void
    {
        $part = $this->createMockPart(contentId: 'test-id@example.com');

        $this->assertTrue($this->strategy->canHandle($part));
    }

    public function testCannotHandleRegularAttachment(): void
    {
        $part = $this->createMockPart(contentId: null);

        $this->assertFalse($this->strategy->canHandle($part));
    }

    public function testGenerateFilenameWithOriginalFilename(): void
    {
        $part = $this->createMockPart(
            contentId: 'test-id@example.com',
            filename: 'original.png',
        );

        $filename = $this->strategy->generateFilename($part);

        $this->assertSame('original.png', $filename);
    }

    public function testGenerateFilenameFromContentId(): void
    {
        $part = $this->createMockPart(
            contentId: 'qr@domain.com',
            filename: null,
            contentType: 'image/png',
        );

        $filename = $this->strategy->generateFilename($part);

        $this->assertSame('qr_domain.com.png', $filename);
    }

    public function testGenerateFilenameWithComplexContentId(): void
    {
        $part = $this->createMockPart(
            contentId: '<logo-embeddable@test.buggregator>',
            filename: null,
            contentType: 'image/svg+xml',
        );

        $filename = $this->strategy->generateFilename($part);

        $this->assertSame('logo-embeddable_test.buggregator.svg', $filename);
    }

    public function testGenerateFilenameWithUnknownMimeType(): void
    {
        $part = $this->createMockPart(
            contentId: 'test-id',
            filename: null,
            contentType: 'application/unknown',
        );

        $filename = $this->strategy->generateFilename($part);

        $this->assertSame('test-id.bin', $filename);
    }

    public function testGenerateFilenameWithEmptyContentId(): void
    {
        $part = $this->createMockPart(
            contentId: '',
            filename: null,
            contentType: 'image/png',
        );

        $filename = $this->strategy->generateFilename($part);

        $this->assertStringStartsWith('inline_', $filename);
        $this->assertStringEndsWith('.png', $filename);
    }

    public function testExtractMetadata(): void
    {
        $part = $this->createMockPart(
            contentId: 'test-id@example.com',
            filename: 'test.png',
            disposition: 'inline',
        );

        $metadata = $this->strategy->extractMetadata($part);

        $this->assertArrayHasKey('content_id', $metadata);
        $this->assertArrayHasKey('is_inline', $metadata);
        $this->assertArrayHasKey('disposition', $metadata);
        $this->assertArrayHasKey('original_filename', $metadata);

        $this->assertSame('test-id@example.com', $metadata['content_id']);
        $this->assertTrue($metadata['is_inline']);
        $this->assertSame('inline', $metadata['disposition']);
        $this->assertSame('test.png', $metadata['original_filename']);
    }

    public function testShouldStoreInline(): void
    {
        $part = $this->createMockPart(contentId: 'test-id');

        $this->assertTrue($this->strategy->shouldStoreInline($part));
    }

    public function testGetPriority(): void
    {
        $this->assertSame(100, $this->strategy->getPriority());
    }

    private function createMockPart(
        ?string $contentId = null,
        ?string $filename = null,
        string $contentType = 'application/octet-stream',
        string $disposition = 'inline',
    ): IMessagePart {
        $part = $this->createMock(IMessagePart::class);
        $part->method('getContentId')->willReturn($contentId);
        $part->method('getFilename')->willReturn($filename);
        $part->method('getContentType')->willReturn($contentType);
        $part->method('getContentDisposition')->willReturn($disposition);

        return $part;
    }
}
