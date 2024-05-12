<?php

declare(strict_types=1);

namespace App\Application\Commands;

use Modules\Webhooks\Domain\Webhook;
use Spiral\Cqrs\QueryInterface;

/**
 * @implements QueryInterface<Webhook[]>
 */
final readonly class FindWebhooks implements QueryInterface {}
