<?php

declare(strict_types=1);

namespace Modules\Webhooks\Application\Locator;

final readonly class YamlFileWebhookLocator implements WebhookLocatorInterface
{
    public function __construct(
        private WebhookFilesFinderInterface $finder,
    ) {}

    public function findAll(): iterable
    {
        yield from $this->finder->find();
    }
}
