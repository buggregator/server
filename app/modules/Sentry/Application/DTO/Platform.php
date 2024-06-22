<?php

declare(strict_types=1);

namespace Modules\Sentry\Application\DTO;

enum Platform: string
{
    case Unknown = 'unknown';
    case Python = 'python';
    case Ruby = 'ruby';
    case PHP = 'php';
    case Laravel = 'laravel';
    case Symfony = 'symfony';
    case Javascript = 'javascript';
    case VueJs = 'vuejs';
    case React = 'react';
    case Angular = 'angular';

    public static function detect(?string $name): self
    {
        if ($name === null) {
            return self::Unknown;
        }

        $name = \strtolower($name);

        return match (true) {
            \str_contains($name, 'python') => self::Python,
            \str_contains($name, 'ruby') => self::Ruby,
            \str_contains($name, 'laravel') => self::Laravel,
            \str_contains($name, 'symfony') => self::Symfony,
            \str_contains($name, 'php') => self::PHP,
            \str_contains($name, 'vue') => self::VueJs,
            \str_contains($name, 'react') => self::React,
            \str_contains($name, 'angular') => self::React,
            \str_contains($name, 'javascript') => self::Javascript,
            default => self::Unknown,
        };
    }
}
