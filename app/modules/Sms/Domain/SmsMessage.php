<?php

declare(strict_types=1);

namespace Modules\Sms\Domain;

use JsonSerializable;

final readonly class SmsMessage implements JsonSerializable
{
    /**
     * @param string[] $warnings Validation warnings (missing required fields for the gateway)
     */
    public function __construct(
        public string $from,
        public string $to,
        public string $message,
        public string $gateway,
        public array $warnings = [],
    ) {}

    public function jsonSerialize(): array
    {
        $data = [
            'from' => $this->from,
            'to' => $this->to,
            'message' => $this->message,
            'gateway' => $this->gateway,
        ];

        if ($this->warnings !== []) {
            $data['warnings'] = $this->warnings;
        }

        return $data;
    }
}
