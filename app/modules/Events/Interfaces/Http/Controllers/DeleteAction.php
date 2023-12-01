<?php

declare(strict_types=1);

namespace Modules\Events\Interfaces\Http\Controllers;

use App\Application\Commands\DeleteEvent;
use App\Application\Domain\ValueObjects\Uuid;
use App\Application\HTTP\Response\ResourceInterface;
use App\Application\HTTP\Response\SuccessResource;
use Spiral\Cqrs\CommandBusInterface;
use Spiral\Router\Annotation\Route;

final class DeleteAction
{
    #[Route(route: 'event/<uuid>', name: 'event.delete', methods: 'DELETE', group: 'api')]
    public function __invoke(CommandBusInterface $bus, Uuid $uuid): ResourceInterface
    {
        $bus->dispatch(
            new DeleteEvent($uuid),
        );

        return new SuccessResource();
    }
}
