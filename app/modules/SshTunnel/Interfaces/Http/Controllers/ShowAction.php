<?php

declare(strict_types=1);

namespace Modules\SshTunnel\Interfaces\Http\Controllers;

use App\Application\Commands\FindSshConnectionByUuid;
use App\Application\Domain\ValueObjects\Uuid;
use App\Application\Exception\EntityNotFoundException;
use App\Application\HTTP\Response\ResourceInterface;
use Modules\SshTunnel\Interfaces\Http\Resources\SshConnectionResource;
use Spiral\Cqrs\QueryBusInterface;
use Spiral\Http\Exception\ClientException\NotFoundException;
use Spiral\Router\Annotation\Route;

final class ShowAction
{
    #[Route(route: 'ssh/<uuid>', name: 'ssh.show', methods: 'GET', group: 'api')]
    public function __invoke(QueryBusInterface $bus, Uuid $uuid): ResourceInterface
    {
        try {
            return new SshConnectionResource(
                $bus->ask(
                    new FindSshConnectionByUuid(
                        uuid: $uuid,
                    ),
                ),
            );
        } catch (EntityNotFoundException $e) {
            throw new NotFoundException($e->getMessage());
        }
    }
}
