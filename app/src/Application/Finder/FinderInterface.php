<?php

declare(strict_types=1);

namespace App\Application\Finder;

interface FinderInterface
{
    /**
     * @return iterable<\SplFileInfo>
     */
    public function find(): iterable;
}
