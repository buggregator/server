<?php

declare(strict_types=1);

namespace Modules\SshTunnel\Application;

use App\Application\Domain\ValueObjects\Uuid;
use Spiral\Exceptions\ExceptionReporterInterface;
use Spiral\RoadRunner\Services\Exception\ServiceException;
use Spiral\RoadRunner\Services\Manager;

final class SshTunnelService
{
    private const NAME_PREFIX = 'ssh-';

    public function __construct(
        private readonly Manager $manager,
        private readonly ExceptionReporterInterface $reporter,
    ) {
    }

    /**
     * Check if the tunnel is connected.
     */
    public function isRunning(Uuid $connectionUuid): bool
    {
        return $this->manager->statuses($this->getTunnelName($connectionUuid)) !== [];
    }

    /**
     * Get the list of all connected tunnels.
     */
    public function list(): array
    {
        return \array_map(
            callback: fn(string $name) => $this->manager->statuses($name),
            array: \array_filter(
                $this->manager->list(),
                static fn(string $service) => \str_starts_with($service, self::NAME_PREFIX),
            ),
        );
    }

    /**
     * Connect to the remote server.
     */
    public function connect(Uuid $connectionUuid): void
    {
        $this->manager->create(
            name: $this->getTunnelName($connectionUuid),
            command: \sprintf('php app.php ssh:tunnel %s', $connectionUuid),
            processNum: 1,
            remainAfterExit: true,
            serviceNameInLogs: true,
        );
    }

    /**
     * Disconnect from the remote server.
     */
    public function disconnect(Uuid $connectionUuid): void
    {
        try {
            $this->manager->terminate($this->getTunnelName($connectionUuid));
        } catch (ServiceException $e) {
            $this->reporter->report($e);
        }
    }

    private function getTunnelName(Uuid $connectionUuid): string
    {
        return self::NAME_PREFIX . $connectionUuid;
    }
}
