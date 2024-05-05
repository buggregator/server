<?php

declare(strict_types=1);

namespace App\Interfaces\Http\Handler;

final readonly class Content implements \Stringable
{
    public int $len;
    public string $contentType;

    public function __construct(
        public string $content,
        string $mime,
    ) {
        $this->contentType = $mime . '; charset=utf-8';
        $this->len = \strlen($content);
    }

    public function __toString(): string
    {
        return $this->content;
    }
}
