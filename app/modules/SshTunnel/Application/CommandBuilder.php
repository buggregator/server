<?php

declare(strict_types=1);

namespace Modules\SshTunnel\Application;

use Spiral\Files\FilesInterface;
use Webmozart\Assert\Assert;

final class CommandBuilder
{
    private const SSH_SERVER_ALIVE_INTERVAL = 15;

    private int $sshPort = 22;
    private string $user = 'root';
    private ?string $sshHost = null;
    private ?string $password = null;
    private ?string $privateKey = null;
    private array $forwardPorts = [];
    private bool $compression = false;

    public function __construct(
        private readonly FilesInterface $files,
    ) {
    }

    public function host(string $host): self
    {
        Assert::notEmpty($host, 'Host cannot be empty');
        $this->sshHost = $host;

        return $this;
    }

    public function user(string $user): self
    {
        Assert::notEmpty($user, 'User cannot be empty');
        $this->user = $user;

        return $this;
    }

    public function sshPort(int $port): self
    {
        Assert::greaterThanEq($port, 1, 'SSH port cannot be less than 1');
        Assert::lessThan($port, 65536, 'SSH port cannot be greater than 65535');

        $this->sshPort = $port;

        return $this;
    }

    public function password(string $password): self
    {
        Assert::notEmpty($password, 'Password cannot be empty');
        $this->password = $password;

        return $this;
    }

    public function privateKey(string $privateKey): self
    {
        Assert::notEmpty($privateKey, 'Private key cannot be empty');
        $this->privateKey = $this->writeKeyToFile($privateKey);
        \usleep(500_000);

        return $this;
    }

    public function forwardPort(int $localPort, int $remotePort): self
    {
        $this->forwardPorts[] = [
            'localPort' => $localPort,
            'remotePort' => $remotePort,
        ];

        return $this;
    }

    public function build(): array
    {
        Assert::true($this->password !== null || $this->privateKey !== null, 'Password or private key must be set');

        $cmd = [
            'ssh',
            '-p',
            $this->sshPort,
            ...\array_map(
                callback: fn(array $port) => \sprintf(
                    '-R %d:%s:%d',
                    $port['localPort'],
                    'localhost',
                    $port['remotePort'],
                ),
                array: $this->forwardPorts,
            ),
            '-i',
            $this->privateKey,
            '-N',
            '-o',
            \sprintf('ServerAliveInterval=%d', self::SSH_SERVER_ALIVE_INTERVAL),
            '-o',
            'ExitOnForwardFailure=yes',
            '-o',
            'StrictHostKeyChecking=no',
            \sprintf('%s@%s', $this->user, $this->sshHost),
        ];

        if ($this->compression) {
            $cmd[] = '-C';
        }

        return $cmd;
    }

    private function writeKeyToFile(string $key): string
    {
        $fileName = (string)\tempnam('/tmp/', 'ssh-key-');

        $this->files->append($fileName, $key);
        $this->files->setPermissions($fileName, 0400);

        return (string)\realpath($fileName);
    }

    public function getPrivateKey(): ?string
    {
        return $this->privateKey;
    }
}
