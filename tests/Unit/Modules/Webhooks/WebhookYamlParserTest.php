<?php

declare(strict_types=1);

namespace Tests\Unit\Modules\Webhooks;

use Modules\Webhooks\Application\Locator\WebhookYamlParser;
use PHPUnit\Framework\Attributes\DataProvider;
use Symfony\Component\Yaml\Parser;
use Tests\TestCase;

final class WebhookYamlParserTest extends TestCase
{
    private const YAML = <<<YAML
webhook:
  event: sentry.received
  url: http://http-dump@buggregator.localhost/webhook
  verify_ssl: false
  retry_on_failure: true
YAML;

    private \Mockery\MockInterface|Parser $yamlParser;
    private WebhookYamlParser $parser;

    protected function setUp(): void
    {
        parent::setUp();

        $this->yamlParser = \Mockery::mock(Parser::class);
        $this->parser = new WebhookYamlParser($this->yamlParser);
    }

    public function testParseValidData(): void
    {
        $this->yamlParser->shouldReceive('parse')
            ->with(self::YAML)
            ->once()
            ->andReturn(
                $parsedData = [
                    'webhook' => [
                        'event' => 'sentry.received',
                        'url' => 'http://http-dump@buggregator.localhost/webhook',
                        'verify_ssl' => false,
                        'retry_on_failure' => true,
                    ],
                ],
            );

        $this->assertSame($parsedData, $this->parser->parse(self::YAML));
    }


    public static function invalidDataProvider(): iterable
    {
        yield 'missing webhook' => [[], 'Missing "webhook" key'];
        yield 'invalid webhook' => [['webhook' => 'invalid'], 'Invalid "webhook" key'];
        yield 'missing event' => [['webhook' => ['url' => 'http://example.com']], 'Missing "event" key'];
        yield 'invalid event' => [['webhook' => ['event' => 42, 'url' => 'http://example.com']], 'Invalid "event" key'];
        yield 'missing url' => [['webhook' => ['event' => 'sentry.received']], 'Missing "url" key'];
        yield 'invalid url' => [['webhook' => ['event' => 'sentry.received', 'url' => 42]], 'Invalid "url" key'];

        yield 'invalid headers' => [
            [
                'webhook' => [
                    'event' => 'sentry.received',
                    'url' => 'http://example.com',
                    'headers' => 'invalid',
                ],
            ],
            'Invalid "headers" key',
        ];

        yield 'invalid header key' => [
            [
                'webhook' => [
                    'event' => 'sentry.received',
                    'url' => 'http://example.com',
                    'headers' => [
                        42 => 'value',
                    ],
                ],
            ],
            'Invalid header key/value',
        ];

        yield 'invalid header value' => [
            [
                'webhook' => [
                    'event' => 'sentry.received',
                    'url' => 'http://example.com',
                    'headers' => [
                        'name' => 42,
                    ],
                ],
            ],
            'Invalid header key/value',
        ];
    }

    #[DataProvider('invalidDataProvider')]
    public function testParseInvalid(array $invalid, string $error): void
    {
        $this->expectException(\RuntimeException::class);
        $this->expectExceptionMessage($error);

        $this->yamlParser->shouldReceive('parse')
            ->with(self::YAML)
            ->once()
            ->andReturn($invalid);

        $this->parser->parse(self::YAML);
    }
}
