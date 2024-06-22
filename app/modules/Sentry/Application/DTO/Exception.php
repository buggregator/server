<?php

declare(strict_types=1);

namespace Modules\Sentry\Application\DTO;

final readonly class Exception
{
    public function __construct(private array $exception) {}

    public function message(): ?string
    {
        return $this->exception['value'] ?? null;
    }

    public function type(): ?string
    {
        return $this->exception['type'] ?? null;
    }

    public function calculateFingerprint(): string
    {
        $string = $this->message() . $this->type();

        foreach ($this->exception['stacktrace']['frames'] as $frame) {
            $string .= $frame['filename'] . $frame['lineno'] . ($frame['context_line'] ?? '');
        }

        return \md5($string);
    }
}
