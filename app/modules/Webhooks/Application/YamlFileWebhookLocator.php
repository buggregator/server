<?php

declare(strict_types=1);

namespace Modules\Webhooks\Application;

use Modules\Webhooks\Domain\WebhookFactoryInterface;
use Modules\Webhooks\Domain\WebhookLocatorInterface;
use Psr\Log\LoggerInterface;
use Symfony\Component\Finder\Finder;
use Symfony\Component\Yaml\Yaml;

final readonly class YamlFileWebhookLocator implements WebhookLocatorInterface
{
    public function __construct(
        private Finder $finder,
        private LoggerInterface $logger,
        private WebhookFactoryInterface $webhookFactory,
        private string $directory,
    ) {
    }

    public function findAll(): iterable
    {
        $this->finder->files()->in($this->directory)->name('*.webhook.yaml');

        foreach ($this->finder as $file) {
            try {
                /**
                 * @var array{webhook: array{
                 *     event: string,
                 *     url: string,
                 *     verify_ssl: bool,
                 *     retry_on_failure: bool,
                 * }} $data
                 */
                $data = Yaml::parseFile($file->getPathname());
                $this->validateWebhookData($data);

                $webhook = $this->webhookFactory->create(
                    event: $data['webhook']['event'],
                    url: $data['webhook']['url'],
                    verifySsl: (bool)($data['webhook']['verify_ssl'] ?? false),
                    retryOnFailure: (bool)($data['webhook']['retry_on_failure'] ?? true),
                );

                foreach ($data['webhook']['headers'] ?? [] as $name => $value) {
                    $webhook = $webhook->withHeader($name, $value);
                }

                yield $webhook;
            } catch (\Throwable $e) {
                $this->logger->error(
                    'Failed to parse webhook file',
                    ['file' => $file->getPathname(), 'error' => $e->getMessage()],
                );
            }
        }
    }

    private function validateWebhookData(array $data): void
    {
        $errors = [];
        // Yaml structure validation
        if (!isset($data['webhook'])) {
            throw new \RuntimeException('Missing "webhook" key');
        }

        if (!\is_array($data['webhook'])) {
            throw new \RuntimeException('Invalid "webhook" key');
        }

        if (!isset($data['webhook']['event'])) {
            $errors[] = 'Missing "event" key';
        } elseif (!\is_string($data['webhook']['event'])) {
            $errors[] = 'Invalid "event" key';
        }

        if (!isset($data['webhook']['url'])) {
            $errors[] = 'Missing "url" key';
        } elseif (!\is_string($data['webhook']['url'])) {
            $errors[] = 'Invalid "url" key';
        } elseif (!\filter_var($data['webhook']['url'], FILTER_VALIDATE_URL)) {
            $errors[] = 'Invalid "url" format';
        }

        // verify headers
        if (isset($data['webhook']['headers']) && !\is_array($data['webhook']['headers'])) {
            $errors[] = 'Invalid "headers" key';

            foreach ($data['webhook']['headers'] as $name => $value) {
                if (!\is_string($name) || !\is_string($value)) {
                    $errors[] = 'Invalid header key/value';
                }
            }
        }

        if ($errors !== []) {
            throw new \RuntimeException(\implode(', ', $errors));
        }
    }
}
