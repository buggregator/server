<?php

declare(strict_types=1);

namespace Modules\Sms\Application\Gateway;

use Modules\Sms\Domain\SmsMessage;

interface GatewayInterface
{
    public function name(): string;

    /**
     * Check if the given payload matches this gateway's signature.
     */
    public function detect(array $body): bool;

    /**
     * Parse the payload into an SmsMessage.
     */
    public function parse(array $body): SmsMessage;

    /**
     * Validate the payload has all required fields.
     * Returns list of missing field names, empty array if valid.
     *
     * @return string[]
     */
    public function validate(array $body): array;
}
