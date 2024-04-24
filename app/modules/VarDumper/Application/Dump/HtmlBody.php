<?php

declare(strict_types=1);

namespace Modules\VarDumper\Application\Dump;

final readonly class HtmlBody implements BodyInterface
{
    private string $id;
    private string $html;

    public function __construct(
        string $value,
    ) {
        $this->html = $value;
        \preg_match_all('/sf-dump-\d+/', $value, $matches);
        $this->id = $matches[0][0];
    }

    public function getId(): string
    {
        return $this->id;
    }

    public function getType(): string
    {
        return 'html';
    }

    public function getValue(): string
    {
        return $this->html;
    }

    public function __toString(): string
    {
        return $this->html;
    }

    public function jsonSerialize(): string
    {
        return $this->__toString();
    }
}
