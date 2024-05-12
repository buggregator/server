<?php

declare(strict_types=1);

namespace Modules\Webhooks\Application\Locator;

use App\Application\Finder\FinderInterface;
use Psr\Log\LoggerInterface;
use Spiral\Files\FilesInterface;

final readonly class WebhookFilesFinder implements WebhookFilesFinderInterface
{
    public function __construct(
        private FinderInterface $finder,
        private LoggerInterface $logger,
        private WebhookYamlParser $parser,
        private FilesInterface $files,
    ) {}

    /**
     * @return iterable<Webhook>
     */
    public function find(): iterable
    {
        foreach ($this->finder->find() as $file) {
            try {
                $data = $this->parser->parse(
                    $this->files->read($file->getPathname()),
                );
            } catch (\Throwable $e) {
                $this->logger->error(
                    'Failed to parse webhook file',
                    ['file' => $file->getPathname(), 'error' => $e->getMessage()],
                );

                continue;
            }

            $webhook = new Webhook(
                key: $file->getBasename('.webhook.yaml'),
                event: $data['webhook']['event'],
                url: $data['webhook']['url'],
                verifySsl: (bool) ($data['webhook']['verify_ssl'] ?? false),
                retryOnFailure: (bool) ($data['webhook']['retry_on_failure'] ?? true),
            );

            foreach ($data['webhook']['headers'] ?? [] as $name => $value) {
                $webhook = $webhook->withHeader($name, $value);
            }

            yield $webhook;
        }
    }
}
