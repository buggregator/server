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
        // Handle escaped periods at the beginning of lines per SMTP spec
        $safeBody = \preg_replace("/^(\.\.)/m", '.', $body);

        // Ensure body is properly appended even with multi-byte characters
        $this->body .= $safeBody;
    }

    public function bodyHasEos(): bool
    {
        // More robust check for end of stream marker
        // This handles potential encoding issues with multi-byte characters
        return \mb_substr($this->body, -5) === "\r\n.\r\n";
    }

    public function getBody(): string
    {
        // Remove the end of stream marker in a way that's safe for multi-byte strings
        if ($this->bodyHasEos()) {
            return \mb_substr($this->body, 0, \mb_strlen($this->body) - 5);
        }

        return $this->body;
    }

    public function parse(): \Modules\Smtp\Application\Mail\Message
    {
        return ParserFactory::getInstance()->create()->parse($this->getBody(), $this->recipients);
    }
}
