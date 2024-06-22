<?php

declare(strict_types=1);

namespace Modules\Sentry\Domain;

use App\Application\Domain\ValueObjects\Uuid;
use Modules\Sentry\Application\DTO\Exception;

interface FingerprintFactoryInterface
{
    /**
     * @param Exception[] $exceptions
     */
    public function create(Uuid $issueUuid, array $exceptions): Fingerprint;
}
