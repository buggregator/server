<?php

declare(strict_types=1);

namespace Modules\Events\Interfaces\Http\Controllers;

use App\Application\Domain\ValueObjects\Uuid;
use Modules\Events\Domain\EventRepositoryInterface;
use Spiral\Http\Exception\ClientException\NotFoundException;
use Spiral\Router\Annotation\Route;

final readonly class PinEventAction
{
    #[Route(route: 'event/<uuid>/pin', name: 'event.pin', methods: ['POST'], group: 'api')]
    public function pin(
        EventRepositoryInterface $events,
        Uuid $uuid,
    ): array {
        if (!$events->pin((string) $uuid)) {
            throw new NotFoundException('Event not found');
        }

        return ['status' => 'pinned'];
    }

    #[Route(route: 'event/<uuid>/pin', name: 'event.unpin', methods: ['DELETE'], group: 'api')]
    public function unpin(
        EventRepositoryInterface $events,
        Uuid $uuid,
    ): array {
        if (!$events->unpin((string) $uuid)) {
            throw new NotFoundException('Event not found');
        }

        return ['status' => 'unpinned'];
    }
}
