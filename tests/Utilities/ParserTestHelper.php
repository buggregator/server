<?php

declare(strict_types=1);

namespace Tests\Utilities;

use Modules\Smtp\Application\Mail\Message;
use Modules\Smtp\Application\Mail\Parser;
use Modules\Smtp\Application\Storage\ParserFactory;

/**
 * Helper class for testing code that uses the Parser class
 */
final class ParserTestHelper
{
    /**
     * Set up a Parser that will return a predetermined Message
     *
     * @param Message $message The Message to return
     * @return Parser The configured Parser
     */
    public static function setupParserWithPredefinedResult(Message $message): Parser
    {
        // Create a real Parser for the test
        $parser = new Parser();

        // Set the test result using reflection since the property is private
        $reflectionClass = new \ReflectionClass(Parser::class);
        $reflectionProperty = $reflectionClass->getProperty('testResult');
        $reflectionProperty->setAccessible(true);
        $reflectionProperty->setValue($parser, $message);

        // Configure the ParserFactory to use our configured Parser
        $parserFactory = ParserFactory::getInstance();
        $parserFactory->setParser($parser);

        return $parser;
    }

    /**
     * Reset the ParserFactory to its default state
     */
    public static function resetParserFactory(): void
    {
        ParserFactory::reset();
    }
}
