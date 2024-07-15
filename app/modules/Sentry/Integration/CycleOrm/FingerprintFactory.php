<?php

declare(strict_types=1);

namespace Modules\Sentry\Integration\CycleOrm;

use App\Application\Domain\ValueObjects\Uuid;
use Modules\Sentry\Domain\Fingerprint;
use Modules\Sentry\Domain\FingerprintFactoryInterface;

final class FingerprintFactory implements FingerprintFactoryInterface
{
    public function create(
        Uuid $issueUuid,
        array $exceptions,
    ): Fingerprint {
        $fingerprints = [];
        foreach ($exceptions as $exception) {
            $fingerprints[] = $exception->calculateFingerprint();
        }

        return new Fingerprint(
            uuid: Uuid::generate(),
            issueUuid: $issueUuid,
            fingerprint: \md5(\implode('', $fingerprints)),
        );
    }
}
