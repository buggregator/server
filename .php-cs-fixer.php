<?php

declare(strict_types=1);

if (!\file_exists(__DIR__ . '/app')) {
    exit(0);
}

return (new PhpCsFixer\Config())
    ->setRules([
        '@PER-CS2.0' => true,
    ])
    ->setRiskyAllowed(true)
    ->setFinder(
        (new PhpCsFixer\Finder())
            ->files()
            ->name('*.php')
            ->in([__DIR__ . '/app/src', __DIR__ . '/app/modules', __DIR__ . '/tests']),
    )
    ->setCacheFile('.cache/.php-cs-fixer.cache');
