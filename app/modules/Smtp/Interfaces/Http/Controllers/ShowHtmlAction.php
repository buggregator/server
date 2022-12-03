<?php

declare(strict_types=1);

namespace Modules\Smtp\Interfaces\Http\Controllers;

use App\Application\Commands\FindEventByUuid;
use App\Application\Domain\ValueObjects\Uuid;
use App\Application\Exception\EntityNotFoundException;
use Modules\Events\Domain\Event;
use Spiral\Cqrs\QueryBusInterface;
use Spiral\Http\Exception\ClientException\NotFoundException;
use Spiral\Http\Exception\ClientException\ServerErrorException;
use Spiral\Router\Annotation\Route;

final class ShowHtmlAction
{
    #[Route(route: '/mail/<uuid>/html', name: 'smtp.show.html')]
    public function __invoke(QueryBusInterface $bus, Uuid $uuid): ?string
    {
        try {
            /** @var Event $event */
            $event = $bus->ask(new FindEventByUuid($uuid));
        } catch (EntityNotFoundException $e) {
            throw new NotFoundException($e->getMessage());
        }

        return $event->getPayload()->jsonSerialize()['html']
            ?? throw new ServerErrorException('No html found in event payload.');
    }
}
