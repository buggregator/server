<?php

declare(strict_types=1);

namespace App\Application\Commands;

use App\Application\Domain\ValueObjects\Uuid;
use Modules\Webhooks\Domain\Delivery;
use Spiral\Cqrs\QueryInterface;

/**
 * @implements QueryInterface<Delivery[]>
 */
final readonly class FindWebhookDeliveries implements QueryInterface
{
    public function __construct(
        public Uuid $webhookUuid,
    ) {}
}
