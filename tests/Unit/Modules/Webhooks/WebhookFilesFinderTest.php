<?php

declare(strict_types=1);

namespace Tests\Unit\Modules\Webhooks;

use App\Application\Finder\FinderInterface;
use Mockery as m;
use Modules\Webhooks\Application\Locator\Webhook;
use Modules\Webhooks\Application\Locator\WebhookFilesFinder;
use Modules\Webhooks\Application\Locator\WebhookYamlParser;
use Psr\Log\LoggerInterface;
use Spiral\Files\FilesInterface;
use Tests\TestCase;

final class WebhookFilesFinderTest extends TestCase
{
    private FinderInterface|m\MockInterface $initialFinder;

    private LoggerInterface|m\MockInterface $logger;

    private WebhookYamlParser|m\MockInterface $parser;

    private FilesInterface|m\MockInterface $files;

    private WebhookFilesFinder $finder;


    protected function setUp(): void
    {
        parent::setUp();

        $this->initialFinder = m::mock(FinderInterface::class);
        $this->logger = m::mock(LoggerInterface::class);
        $this->parser = m::mock(WebhookYamlParser::class);
        $this->files = m::mock(FilesInterface::class);

        $this->finder = new WebhookFilesFinder(
            finder: $this->initialFinder,
            logger: $this->logger,
            parser: $this->parser,
            files: $this->files,
        );
    }

    public function testFindFiles(): void
    {
        $this->initialFinder->shouldReceive('find')
            ->once()
            ->andReturn([
                $file1 = m::mock(\SplFileInfo::class),
                $file2 = m::mock(\SplFileInfo::class),
            ]);

        $file1->shouldReceive('getPathname')
            ->andReturn($file1Name = 'file1.txt');

        $file1->shouldReceive('getBasename')
            ->with('.webhook.yaml')
            ->andReturn('file1');

        $file2->shouldReceive('getPathname')
            ->twice()
            ->andReturn($file2Name = 'file2.txt');

        $this->files->shouldReceive('read')
            ->once()
            ->with($file1Name)
            ->andReturn($file1Content = 'file1 content');

        $this->files->shouldReceive('read')
            ->once()
            ->with($file2Name)
            ->andThrow(new \Exception($errorMessage = 'file2 error'));

        $this->logger->shouldReceive('error')
            ->once()
            ->with(
                'Failed to parse webhook file',
                ['file' => $file2Name, 'error' => $errorMessage],
            );

        $this->parser->shouldReceive('parse')
            ->with($file1Content)
            ->once()
            ->andReturn([
                'webhook' => [
                    'event' => 'event1',
                    'url' => 'url1',
                    'verify_ssl' => true,
                    'retry_on_failure' => false,
                    'headers' => [
                        'header1' => 'value1',
                    ],
                ],
            ]);

        $result = \iterator_to_array($this->finder->find());

        $this->assertEquals(
            new Webhook(
                key: 'file1',
                event: 'event1',
                url: 'url1',
                headers: ['header1' => ['value1']],
                verifySsl: true,
                retryOnFailure: false,
            ),
            $result[0],
        );
    }
}
