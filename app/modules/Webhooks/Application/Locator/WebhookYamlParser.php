<?php

declare(strict_types=1);

namespace Modules\Webhooks\Application\Locator;

use Symfony\Component\Yaml\Parser;

/**
 * @psalm-type WebhookData = array{webhook: array{
 *      event: string,
 *      url: string,
 *      verify_ssl: bool,
 *      retry_on_failure: bool,
 *  }}
 */
class WebhookYamlParser
{
    public function __construct(
        private readonly Parser $parser,
    ) {}

    /**
     * @param non-empty-string $yaml
     * @return WebhookData
     */
    public function parse(string $yaml): array
    {
        /**
         * @var WebhookData $data
         */
        $data = $this->parser->parse($yaml);

        $this->validateWebhookData($data);

        return $data;
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
        if (isset($data['webhook']['headers'])) {
            if (!\is_array($data['webhook']['headers'])) {
                $errors[] = 'Invalid "headers" key';
            } else {
                foreach ($data['webhook']['headers'] as $name => $value) {
                    if (!\is_string($name) || !\is_string($value)) {
                        $errors[] = 'Invalid header key/value';
                    }
                }
            }
        }

        if ($errors !== []) {
            throw new \RuntimeException(\implode(', ', $errors));
        }
    }
}
