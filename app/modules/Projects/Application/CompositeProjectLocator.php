<?php

declare(strict_types=1);

namespace Modules\Projects\Application;

use Modules\Projects\Domain\ProjectLocatorInterface;

final readonly class CompositeProjectLocator implements ProjectLocatorInterface
{
    /**
     * @param ProjectLocatorInterface[] $locators
     */
    public function __construct(
        private array $locators,
    ) {
    }

    public function findAll(): iterable
    {
        foreach ($this->locators as $locator) {
            yield from $locator->findAll();
        }
    }
}
