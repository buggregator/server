<?php

declare(strict_types=1);

namespace Tests\Feature\Interfaces\TCP\Smtp;

use Modules\Smtp\Application\Storage\Message;
use Tests\TestCase;

final class MessageTest extends TestCase
{
    public function testCyrillicBodyHandling(): void
    {
        $message = new Message('test-uuid');

        // Standard ASCII content
        $message->appendBody("Hello World\r\n");
        $this->assertFalse($message->bodyHasEos());

        // Add Cyrillic content
        $message->appendBody("съешь же ещё этих мягких французских булок, да выпей чаю\r\n");
        $this->assertFalse($message->bodyHasEos());

        // Add more Cyrillic content with varying lengths
        $cyrillicRepeated = str_repeat("съешь же ещё этих мягких французских булок, да выпей чаю\r\n", 5);
        $message->appendBody($cyrillicRepeated);
        $this->assertFalse($message->bodyHasEos());

        // Add end of stream marker
        $message->appendBody(".\r\n");
        $this->assertTrue($message->bodyHasEos());

        // Check that getBody removes the EOS marker
        $body = $message->getBody();
        $this->assertStringNotContainsString("\r\n.\r\n", $body);

        // Verify content is preserved
        $this->assertStringContainsString("Hello World", $body);
        $this->assertStringContainsString("съешь же ещё этих мягких французских булок, да выпей чаю", $body);
    }

    public function testMultipleEOSMarkers(): void
    {
        $message = new Message('test-uuid');

        // Add content with a period at the start of a line (which should be escaped)
        $message->appendBody("Line 1\r\n");
        $message->appendBody(".Line starting with period\r\n");
        $message->appendBody("Line 3\r\n");

        // Add an EOS marker in the middle (shouldn't be considered as EOS)
        $message->appendBody(".\r\nMore content\r\n");
        $this->assertFalse($message->bodyHasEos());

        // Now add actual EOS marker at the end
        $message->appendBody(".\r\n");
        $this->assertTrue($message->bodyHasEos());

        // Verify the body content is correct
        $body = $message->getBody();
        $this->assertStringContainsString("Line 1", $body);
        $this->assertStringContainsString(".Line starting with period", $body);
        $this->assertStringContainsString("Line 3", $body);
        $this->assertStringContainsString(".\r\nMore content", $body);
    }

    public function testLargeBodyWithEOS(): void
    {
        $message = new Message('test-uuid');

        // Add a large body
        $largeBody = str_repeat("Lorem ipsum dolor sit amet. ", 1000);
        $message->appendBody($largeBody);
        $message->appendBody("\r\n.\r\n");

        $this->assertTrue($message->bodyHasEos());

        // Verify the body content is correct
        $body = $message->getBody();
        $this->assertStringContainsString("Lorem ipsum dolor sit amet.", $body);
        $this->assertStringNotContainsString("\r\n.\r\n", $body);
    }
}
