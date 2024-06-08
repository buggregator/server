<?php

declare(strict_types=1);

namespace App\Application\Database;

interface CleanerInterface
{
    public function clean(?string $database = null): \Generator;
}
