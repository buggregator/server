<?php

declare(strict_types=1);

namespace Modules\Smtp\Application\Storage;

use Modules\Smtp\Application\Mail\Parser;

final class Message
{
    public function __construct(
        public string $uuid,
        public bool $waitUsername = false,
        public bool $waitPassword = false,
        public bool $waitBody = false,
        public ?string $username = null,
        public ?string $password = null,
        public array $recipients = [],
        public ?string $from = null,
        public string $body = '',
    ) {}

    public function setUsername(string $username): void
    {
        $this->username = \base64_decode(\trim($username));
        $this->waitUsername = false;
        $this->waitPassword = true;
    }

    public function setPassword(string $password): void
    {
        $this->password = \base64_decode(\trim($password));
        $this->waitPassword = false;
    }

    public function addRecipient(string $recipient): void
    {
        $this->recipients[] = $recipient;
    }

    public function setFrom(string $from): void
    {
        $this->from = $from;
    }

    public function appendBody(string $body): void
    {
        $this->body .= \preg_replace("/^(\.\.)/m", '.', $body);
    }

    public function bodyHasEos(): bool
    {
        return \str_ends_with($this->body, "\r\n.\r\n");
    }

    public function getBody(): string
    {
        return \str_replace("\r\n.\r\n", '', $this->body);
    }

    public function parse(): \Modules\Smtp\Application\Mail\Message
    {
        return (new Parser())->parse($this->getBody());
    }
}
