<?php

declare(strict_types=1);

namespace Tests\Unit\Modules\Smtp\Application\Mail\Strategy;

use Modules\Smtp\Application\Mail\AttachmentProcessor;
use Modules\Smtp\Application\Mail\Strategy\AttachmentProcessingStrategy;
use Modules\Smtp\Application\Mail\Strategy\AttachmentProcessorFactory;
use Modules\Smtp\Application\Mail\Strategy\InlineAttachmentStrategy;
use Modules\Smtp\Application\Mail\Strategy\RegularAttachmentStrategy;
use Modules\Smtp\Application\Mail\Strategy\FallbackAttachmentStrategy;
use Tests\TestCase;
use ZBateson\MailMimeParser\Message\IMessagePart;

final class AttachmentProcessorFactoryTest extends TestCase
{
    private AttachmentProcessorFactory $factory;

    protected function setUp(): void
    {
        parent::setUp();
        $this->factory = new AttachmentProcessorFactory();
    }

    public function testDetermineStrategyForInlineAttachment(): void
    {
        $part = $this->createMockPart(contentId: 'test-id@example.com');

        $strategy = $this->factory->determineStrategy($part);

        $this->assertInstanceOf(InlineAttachmentStrategy::class, $strategy);
    }

    public function testDetermineStrategyForRegularAttachment(): void
    {
        $part = $this->createMockPart(contentId: null, disposition: 'attachment');

        $strategy = $this->factory->determineStrategy($part);

        $this->assertInstanceOf(RegularAttachmentStrategy::class, $strategy);
    }

    public function testDetermineStrategyForUnknownAttachment(): void
    {
        $part = $this->createMockPart(contentId: null, disposition: 'unknown');

        $strategy = $this->factory->determineStrategy($part);

        // Should fall back to RegularAttachmentStrategy or FallbackAttachmentStrategy
        $this->assertInstanceOf(RegularAttachmentStrategy::class, $strategy);
    }

    public function testCreateProcessor(): void
    {
        $part = $this->createMockPart(contentId: 'test-id@example.com');

        $processor = $this->factory->createProcessor($part);

        $this->assertInstanceOf(AttachmentProcessor::class, $processor);
        $this->assertInstanceOf(InlineAttachmentStrategy::class, $processor->getStrategy());
    }

    public function testGetStrategies(): void
    {
        $strategies = $this->factory->getStrategies();

        $this->assertCount(3, $strategies);
        $this->assertInstanceOf(InlineAttachmentStrategy::class, $strategies[0]);
        $this->assertInstanceOf(RegularAttachmentStrategy::class, $strategies[1]);
        $this->assertInstanceOf(FallbackAttachmentStrategy::class, $strategies[2]);
    }

    public function testRegisterStrategy(): void
    {
        $customStrategy = new class implements AttachmentProcessingStrategy {
            public function canHandle(IMessagePart $part): bool
            {
                return true;
            }

            public function generateFilename(IMessagePart $part): string
            {
                return 'custom.txt';
            }

            public function extractMetadata(IMessagePart $part): array
            {
                return ['custom' => true];
            }

            public function shouldStoreInline(IMessagePart $part): bool
            {
                return false;
            }

            public function getPriority(): int
            {
                return 200;
            }
        };

        $this->factory->registerStrategy($customStrategy);

        $strategies = $this->factory->getStrategies();
        $this->assertCount(4, $strategies);
        $this->assertSame($customStrategy, $strategies[3]);
    }

    public function testStrategyPriorityOrdering(): void
    {
        $part = $this->createMockPart(contentId: 'test-id@example.com');

        // Both InlineAttachmentStrategy and FallbackAttachmentStrategy can handle this
        // but InlineAttachmentStrategy has higher priority
        $strategy = $this->factory->determineStrategy($part);

        $this->assertInstanceOf(InlineAttachmentStrategy::class, $strategy);
        $this->assertSame(100, $strategy->getPriority());
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
