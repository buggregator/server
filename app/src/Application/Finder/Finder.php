<?php

declare(strict_types=1);

namespace App\Application\Finder;

/**
 * todo: give a better name
 */
final readonly class Finder implements FinderInterface
{
    public function __construct(
        private \Symfony\Component\Finder\Finder $finder,
    ) {}

    public function find(): iterable
    {
        yield from $this->finder;
    }
}
