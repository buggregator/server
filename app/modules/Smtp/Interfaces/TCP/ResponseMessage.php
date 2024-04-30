<?php

declare(strict_types=1);

namespace Modules\Smtp\Interfaces\TCP;

final readonly class ResponseMessage implements \Stringable
{
    private const READY = 220;
    public const OK = 250;
    public const CLOSING = 221;
    public const AUTHENTICATED = 235;
    public const START_MAIL_INPUT = 354;
    public const USERNAME = 334;

    public static function ready(string $message = 'mailamie'): self
    {
        return new self(self::READY, $message);
    }

    public static function ok(string $message = ''): self
    {
        return new self(self::OK, $message);
    }

    public static function authRequired(string $message = 'AUTH LOGIN PLAIN CRAM-MD5'): self
    {
        return new self(self::OK, $message);
    }

    public static function closing(string $message = ''): self
    {
        return new self(self::CLOSING, $message);
    }

    public static function authenticated(string $message = 'Authentication successful'): self
    {
        return new self(self::AUTHENTICATED, $message);
    }

    public static function provideBody(string $message = ''): self
    {
        return new self(self::START_MAIL_INPUT, $message);
    }

    public static function enterUsername(string $message = 'Username'): self
    {
        return new self(self::USERNAME, \base64_encode($message));
    }

    public static function enterPassword(string $message = 'Password'): self
    {
        return new self(self::USERNAME, \base64_encode($message));
    }

    public function __construct(
        public int $code,
        public string $message = '',
    ) {}

    public function __toString(): string
    {
        return \sprintf("%d %s\r\n", $this->code, empty($this->message) ? '' : $this->message . ' ');
    }
}
