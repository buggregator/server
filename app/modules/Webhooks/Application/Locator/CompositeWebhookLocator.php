<?php

declare(strict_types=1);

namespace Modules\Webhooks\Application\Locator;

final readonly class CompositeWebhookLocator implements WebhookLocatorInterface
{
    /**
     * @param WebhookLocatorInterface[] $locators
     */
    public function __construct(
        private array $locators,
    ) {}

    public function findAll(): iterable
    {
        foreach ($this->locators as $locator) {
            yield from $locator->findAll();
        }
    }
}
