<?php

declare(strict_types=1);

namespace Modules\Smtp\Application\Storage;

use Modules\Smtp\Application\Mail\Parser;

/**
 * Factory for creating Parser instances
 * This class exists primarily to facilitate testing by allowing
 * the Parser instantiation to be mocked or overridden.
 *
 * @internal
 */
final class ParserFactory
{
    // Static instance for singleton pattern
    private static ?self $instance = null;

    // The Parser instance or null
    private ?Parser $parser = null;

    /**
     * Get the factory instance
     */
    public static function getInstance(): self
    {
        if (!self::$instance instanceof \Modules\Smtp\Application\Storage\ParserFactory) {
            self::$instance = new self();
        }

        return self::$instance;
    }

    /**
     * Reset the factory instance (useful for testing)
     */
    public static function reset(): void
    {
        self::$instance = null;
    }

    /**
     * Set a custom Parser instance
     *
     * @param Parser $parser The Parser instance to use
     */
    public function setParser(Parser $parser): void
    {
        $this->parser = $parser;
    }

    /**
     * Create a Parser instance
     *
     * @return Parser The Parser instance
     */
    public function create(): Parser
    {
        return $this->parser ?? new Parser();
    }
}
