<?php

declare(strict_types=1);

namespace Modules\Sms\Application\Gateway;

use Modules\Sms\Domain\SmsMessage;

/**
 * Config-driven gateway: detects provider by field presence, parses via field mapping.
 * Adding a new provider = creating a new instance with different config.
 */
final readonly class FieldMapGateway implements GatewayInterface
{
    /**
     * @param string $gatewayName Provider name (e.g. 'twilio', 'vonage')
     * @param string[] $detectFields Fields that must ALL be present to identify this provider
     * @param string[] $fromFields Candidate field names for "from" (first non-empty wins)
     * @param string[] $toFields Candidate field names for "to"
     * @param string[] $messageFields Candidate field names for "message"
     */
    public function __construct(
        private string $gatewayName,
        private array $detectFields,
        private array $fromFields,
        private array $toFields,
        private array $messageFields,
    ) {}

    public function name(): string
    {
        return $this->gatewayName;
    }

    public function detect(array $body): bool
    {
        if ($this->detectFields === []) {
            return true;
        }

        foreach ($this->detectFields as $field) {
            if (!isset($body[$field])) {
                return false;
            }
        }

        return true;
    }

    public function validate(array $body): array
    {
        $missing = [];

        foreach ($this->detectFields as $field) {
            if (!isset($body[$field])) {
                $missing[] = $field;
            }
        }

        if ($this->extractFirst($body, $this->toFields) === '') {
            $missing[] = \implode('|', $this->toFields);
        }

        if ($this->extractFirst($body, $this->messageFields) === '') {
            $missing[] = \implode('|', $this->messageFields);
        }

        return $missing;
    }

    public function parse(array $body): SmsMessage
    {
        return new SmsMessage(
            from: $this->extractFirst($body, $this->fromFields),
            to: $this->extractFirst($body, $this->toFields),
            message: $this->extractFirst($body, $this->messageFields),
            gateway: $this->gatewayName,
        );
    }

    /**
     * @param string[] $keys
     */
    private function extractFirst(array $body, array $keys): string
    {
        foreach ($keys as $key) {
            if (isset($body[$key]) && \is_string($body[$key]) && $body[$key] !== '') {
                return $body[$key];
            }
        }

        return '';
    }
}
