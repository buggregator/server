<?php

declare(strict_types=1);

namespace Tests\Feature\Modules\Smtp\Application\Mail;

use Modules\Smtp\Application\Mail\Parser;
use Tests\TestCase;

final class InlineAttachmentParsingTest extends TestCase
{
    private Parser $parser;

    protected function setUp(): void
    {
        parent::setUp();
        $this->parser = $this->get(Parser::class);
    }

    public function testParseEmailWithInlineAttachmentWithoutFilename(): void
    {
        // This simulates the problematic case where an inline attachment has no filename
        $rawEmail = $this->buildRawEmailWithInlineAttachmentWithoutFilename();

        $message = $this->parser->parse($rawEmail);

        // Verify the message was parsed successfully
        $this->assertSame('Test with inline attachment', $message->subject);
        $this->assertCount(1, $message->attachments);

        // Verify the attachment was processed correctly
        $attachment = $message->attachments[0];

        $this->assertSame('qr_domain.com.png', $attachment->getFilename());
        $this->assertSame('image/png', $attachment->getType());
        $this->assertSame('qr@domain.com', $attachment->getContentId());
        $this->assertNotEmpty($attachment->getContent());
    }

    public function testParseEmailWithComplexContentId(): void
    {
        $rawEmail = $this->buildRawEmailWithComplexContentId();

        $message = $this->parser->parse($rawEmail);

        $this->assertCount(1, $message->attachments);
        $attachment = $message->attachments[0];

        // Complex content-id should be sanitized to valid filename
        $this->assertSame('logo-embeddable_test.buggregator.svg', $attachment->getFilename());
        $this->assertSame('image/svg+xml', $attachment->getType());
        $this->assertSame('logo-embeddable@test.buggregator', $attachment->getContentId());
    }

    public function testParseEmailWithMixedAttachments(): void
    {
        $rawEmail = $this->buildRawEmailWithMixedAttachments();

        $message = $this->parser->parse($rawEmail);

        $this->assertCount(2, $message->attachments);

        // First attachment: inline without filename
        $inlineAttachment = $message->attachments[0];
        $this->assertSame('test-id_example.com.jpg', $inlineAttachment->getFilename());
        $this->assertSame('image/jpeg', $inlineAttachment->getType());
        $this->assertSame('test-id@example.com', $inlineAttachment->getContentId());

        // Second attachment: regular with filename
        $regularAttachment = $message->attachments[1];
        $this->assertSame('document.pdf', $regularAttachment->getFilename());
        $this->assertSame('application/pdf', $regularAttachment->getType());
        $this->assertNull($regularAttachment->getContentId());
    }

    private function buildRawEmailWithInlineAttachmentWithoutFilename(): string
    {
        return <<<'EMAIL'
            From: sender@example.com
            To: recipient@example.com
            Subject: Test with inline attachment
            Content-Type: multipart/related; boundary="boundary123"
            
            --boundary123
            Content-Type: text/html; charset=UTF-8
            
            <html>
            <body>
            <p>This is a test with inline attachment:</p>
            <img src="cid:qr@domain.com" alt="QR Code">
            </body>
            </html>
            
            --boundary123
            Content-Type: image/png
            Content-Disposition: inline
            Content-ID: <qr@domain.com>
            
            iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg==
            --boundary123--
            EMAIL;
    }

    private function buildRawEmailWithComplexContentId(): string
    {
        return <<<'EMAIL'
            From: sender@example.com
            To: recipient@example.com
            Subject: Test with complex content-id
            Content-Type: multipart/related; boundary="boundary456"
            
            --boundary456
            Content-Type: text/html; charset=UTF-8
            
            <html>
            <body>
            <p>This is a test with complex content-id:</p>
            <img src="cid:logo-embeddable@test.buggregator" alt="Logo">
            </body>
            </html>
            
            --boundary456
            Content-Type: image/svg+xml
            Content-Disposition: inline
            Content-ID: <logo-embeddable@test.buggregator>
            
            <svg xmlns="http://www.w3.org/2000/svg" width="100" height="100">
              <rect width="100" height="100" fill="red"/>
            </svg>
            --boundary456--
            EMAIL;
    }

    private function buildRawEmailWithMixedAttachments(): string
    {
        return <<<'EMAIL'
            From: sender@example.com
            To: recipient@example.com
            Subject: Test with mixed attachments
            Content-Type: multipart/mixed; boundary="boundary789"
            
            --boundary789
            Content-Type: text/html; charset=UTF-8
            
            <html>
            <body>
            <p>This email has both inline and regular attachments:</p>
            <img src="cid:test-id@example.com" alt="Inline Image">
            <p>And a regular attachment below.</p>
            </body>
            </html>
            
            --boundary789
            Content-Type: image/jpeg
            Content-Disposition: inline
            Content-ID: <test-id@example.com>
            
            /9j/4AAQSkZJRgABAQEAYABgAAD/2wBDAAEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQH/2wBDAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQH/wAARCAABAAEDASIAAhEBAxEB/8QAFQABAQAAAAAAAAAAAAAAAAAAAAv/xAAUEAEAAAAAAAAAAAAAAAAAAAAA/8QAFQEBAQAAAAAAAAAAAAAAAAAAAAX/xAAUEQEAAAAAAAAAAAAAAAAAAAAA/9oADAMBAAIRAxEAPwA/gA==
            
            --boundary789
            Content-Type: application/pdf
            Content-Disposition: attachment; filename="document.pdf"
            
            JVBERi0xLjQKJeLjz9MKMSAwIG9iago8PAovVHlwZSAvQ2F0YWxvZwovUGFnZXMgMiAwIFIKPj4KZW5kb2JqCjIgMCBvYmoKPDwKL1R5cGUgL1BhZ2VzCi9LaWRzIFszIDAgUl0KL0NvdW50IDEKL01lZGlhQm94IFswIDAgNjEyIDc5Ml0KPj4KZW5kb2JqCjMgMCBvYmoKPDwKL1R5cGUgL1BhZ2UKL1BhcmVudCAyIDAgUgo+PgplbmRvYmoKdHJhaWxlcgo8PAovUm9vdCAxIDAgUgo+PgpzdGFydHhyZWYKMjczCiUlRU9G
            --boundary789--
            EMAIL;
    }

    public function testParseEmailWithFallbackStrategy(): void
    {
        // Test edge case that should trigger fallback strategy
        $rawEmail = $this->buildRawEmailWithUnknownAttachmentType();

        $message = $this->parser->parse($rawEmail);

        $this->assertCount(1, $message->attachments);
        $attachment = $message->attachments[0];

        // Should generate a filename since no content-id or filename is provided
        $this->assertStringStartsWith('attachment_', $attachment->getFilename());
        $this->assertStringEndsWith('.bin', $attachment->getFilename());
        $this->assertSame('application/octet-stream', $attachment->getType());
    }

    private function buildRawEmailWithUnknownAttachmentType(): string
    {
        return <<<'EMAIL'
            From: sender@example.com
            To: recipient@example.com
            Subject: Test with unknown attachment
            Content-Type: multipart/mixed; boundary="boundary999"
            
            --boundary999
            Content-Type: text/plain; charset=UTF-8
            
            This email has an unknown attachment type.
            
            --boundary999
            Content-Type: application/octet-stream
            Content-Disposition: attachment
            
            BinaryDataHere123456789
            --boundary999--
            EMAIL;
    }

    public function testParseEmailWithEmptyContentId(): void
    {
        $rawEmail = $this->buildRawEmailWithEmptyContentId();

        $message = $this->parser->parse($rawEmail);

        $this->assertCount(1, $message->attachments);
        $attachment = $message->attachments[0];

        // Should generate a filename when content-id is empty
        $this->assertStringStartsWith('attachment_', $attachment->getFilename());
        $this->assertStringEndsWith('.png', $attachment->getFilename());
        $this->assertSame('image/png', $attachment->getType());
    }

    private function buildRawEmailWithEmptyContentId(): string
    {
        return <<<'EMAIL'
            From: sender@example.com
            To: recipient@example.com
            Subject: Test with empty content-id
            Content-Type: multipart/related; boundary="boundary111"
            
            --boundary111
            Content-Type: text/html; charset=UTF-8
            
            <html>
            <body>
            <p>This has an empty content-id:</p>
            <img src="cid:" alt="Empty CID">
            </body>
            </html>
            
            --boundary111
            Content-Type: image/png
            Content-Disposition: inline
            Content-ID: <>
            
            iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg==
            --boundary111--
            EMAIL;
    }

    public function testParseEmailHandlesExceptionsGracefully(): void
    {
        // Test that parser handles malformed attachments without crashing
        $rawEmail = $this->buildRawEmailWithMalformedAttachment();

        // This should not throw an exception
        $message = $this->parser->parse($rawEmail);

        // Should still parse the message even with malformed attachment
        $this->assertSame('Test with malformed attachment', $message->subject);

        // Attachment processing might fail, but message parsing should continue
        // The exact behavior depends on the underlying mail parser library
    }

    private function buildRawEmailWithMalformedAttachment(): string
    {
        return <<<'EMAIL'
            From: sender@example.com
            To: recipient@example.com
            Subject: Test with malformed attachment
            Content-Type: multipart/mixed; boundary="boundary222"
            
            --boundary222
            Content-Type: text/plain; charset=UTF-8
            
            This email has a malformed attachment.
            
            --boundary222
            Content-Type: image/png
            Content-Disposition: attachment
            Content-ID: <malformed
            
            This is not valid PNG data and has malformed headers
            --boundary222--
            EMAIL;
    }
}
