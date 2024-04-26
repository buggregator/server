<?php

declare(strict_types=1);

namespace Modules\Projects\Application;

use Modules\Projects\Domain\Project;
use Modules\Projects\Domain\ProjectFactoryInterface;
use Modules\Projects\Exception\InvalidKeyFormatException;

final readonly class ProjectFactory implements ProjectFactoryInterface
{
    public function __construct(
        private string $allowedKeyCharacters = 'a-z0-9-_',
    ) {
    }

    public function create(string $key, string $name): Project
    {
        if (\preg_match('/^[' . $this->allowedKeyCharacters . ']+$/', $key) !== 1) {
            throw new InvalidKeyFormatException(
                'Invalid project key. Key must contain only lowercase letters, numbers, hyphens and underscores.',
            );
        }

        return new Project($key, $name);
    }
}
