<?php

declare(strict_types=1);

namespace Modules\Ray\Interfaces\Http\Controllers;

use App\Application\Commands\FindEventByUuid;
use App\Application\Domain\ValueObjects\Uuid;
use App\Application\Exception\EntityNotFoundException;
use Spiral\Cqrs\QueryBusInterface;
use Spiral\Http\Exception\ClientException\NotFoundException;
use Spiral\Http\Exception\ClientException\ServerErrorException;
use Spiral\Router\Annotation\Route;

final class ShowHtmlAction
{
    #[Route(route: '/ray/<uuid>/html', name: 'ray.show.html')]
    public function __invoke(QueryBusInterface $bus, Uuid $uuid): string
    {
        try {
            $event = $bus->ask(new FindEventByUuid($uuid));
        } catch (EntityNotFoundException $e) {
            throw new NotFoundException($e->getMessage());
        }

        $eventData = $event->getPayload()->toArray();

        $type = $eventData['payloads'][0]['type'] ?? null;

        if ($type !== 'mailable') {
            throw new NotFoundException('Event type should be mailable.');
        }

        $html = $eventData['payloads'][0]['content']['html'] ?? null;

        if ($html === null) {
            throw new ServerErrorException('HTML is empty.');
        }

        return $html;
    }
}
