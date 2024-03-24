<?php

declare(strict_types=1);

namespace Modules\SshTunnel\Interfaces\Console;

use Modules\SshTunnel\Application\CommandBuilder;
use Modules\SshTunnel\Domain\ConnectionRepositoryInterface;
use Modules\SshTunnel\Exception\ConnectionException;
use Spiral\Console\Attribute\Argument;
use Spiral\Console\Attribute\AsCommand;
use Spiral\Console\Command;
use Symfony\Component\Console\Command\SignalableCommandInterface;
use Symfony\Component\Process\Process;

#[AsCommand(
    name: 'ssh:tunnel',
    description: 'Create SSH tunnel to remote server',
)]
final class SshTunnelCommand extends Command implements SignalableCommandInterface
{
    #[Argument(description: 'SSH connection UUID')]
    public string $connectionUuid;

    private Process $process;
    private ?string $provateKeyPath = null;

    public function __invoke(ConnectionRepositoryInterface $connections, CommandBuilder $builder): int
    {
        $connection = $connections->findByPK($this->connectionUuid);

        if (!$connection) {
            $this->error('Connection not found');
            return self::FAILURE;
        }

        $command = $builder->host($connection->host)
            ->sshPort($connection->port)
            ->user($connection->user)
            ->privateKey($connection->privateKey)
            ->forwardPort(8000, 8000)
            ->forwardPort(1025, 1025)
            ->forwardPort(9913, 9913)
            ->forwardPort(9912, 9912)
            ->build();

        if ($this->isVerbose()) {
            $this->info('SSH command:');
            $this->comment(\implode(' ', $command));
        }

        $this->provateKeyPath = $builder->getPrivateKey();

        $this->process = new Process(command: $command, timeout: null);
        $this->process->start(function () {
            $this->writeln('SSH tunnel started');
        });

        $this->process->wait(function ($type, $buffer) {
            $this->writeln($buffer);
        });

        return self::SUCCESS;
    }

    public function getSubscribedSignals(): array
    {
        return [
            SIGINT,
            SIGTERM,
        ];
    }

    public function handleSignal(int $signal,): int|false
    {
        $this->process->stop();

        $this->warning('SSH tunnel stopped');

        if ($this->provateKeyPath) {
            unlink($this->provateKeyPath);
            $this->warning('SSH private key deleted');
        }

        return false;
    }
}
