<?php

declare(strict_types=1);

namespace Modules\Sms\Application\Gateway;

/**
 * Tries gateways in order, returns the first match.
 * Returns null if no gateway matched — the handler should pass the request through.
 */
final readonly class GatewayRegistry
{
    /** @param GatewayInterface[] $gateways */
    public function __construct(
        private array $gateways,
    ) {}

    public function detect(array $body): ?GatewayInterface
    {
        foreach ($this->gateways as $gateway) {
            if ($gateway->detect($body)) {
                return $gateway;
            }
        }

        return null;
    }

    public function findByName(string $name): ?GatewayInterface
    {
        foreach ($this->gateways as $gateway) {
            if ($gateway->name() === $name) {
                return $gateway;
            }
        }

        return null;
    }
}
