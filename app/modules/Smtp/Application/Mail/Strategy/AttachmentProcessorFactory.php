<?php

declare(strict_types=1);

namespace Modules\Smtp\Application\Mail\Strategy;

use Modules\Smtp\Application\Mail\AttachmentProcessor;
use ZBateson\MailMimeParser\Message\IMessagePart;

final class AttachmentProcessorFactory
{
    /**
     * @param AttachmentProcessingStrategy[] $strategies
     */
    public function __construct(
        private array $strategies = [
            new InlineAttachmentStrategy(),
            new RegularAttachmentStrategy(),
            new FallbackAttachmentStrategy(),
        ],
    ) {}

    /**
     * Determines the appropriate strategy for processing the given message part
     */
    public function determineStrategy(IMessagePart $part): AttachmentProcessingStrategy
    {
        $availableStrategies = \array_filter(
            $this->strategies,
            static fn(AttachmentProcessingStrategy $strategy) => $strategy->canHandle($part),
        );

        if ($availableStrategies === []) {
            // This should never happen due to FallbackAttachmentStrategy
            throw new \RuntimeException('No strategy available to handle the message part');
        }

        // Sort by priority (highest first)
        \usort($availableStrategies, fn($a, $b) => $b->getPriority() <=> $a->getPriority());

        return $availableStrategies[0];
    }

    /**
     * Creates a processor with the appropriate strategy for the given part
     */
    public function createProcessor(IMessagePart $part): AttachmentProcessor
    {
        $strategy = $this->determineStrategy($part);
        return new AttachmentProcessor($strategy);
    }

    /**
     * Registers a custom strategy
     */
    public function registerStrategy(AttachmentProcessingStrategy $strategy): void
    {
        $this->strategies[] = $strategy;
    }

    /**
     * Gets all registered strategies
     *
     * @return AttachmentProcessingStrategy[]
     */
    public function getStrategies(): array
    {
        return $this->strategies;
    }
}
