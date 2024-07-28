<?php

declare(strict_types=1);

namespace App\Interfaces\Centrifugo;

use RoadRunner\Centrifugo\Request\Connect;
use App\Application\Broadcasting\Channel\EventsChannel;
use App\Application\Commands\FindAllProjects;
use Modules\Projects\Domain\Project;
use RoadRunner\Centrifugo\Payload\ConnectResponse;
use RoadRunner\Centrifugo\Request\RequestInterface;
use Spiral\Cqrs\QueryBusInterface;
use Spiral\RoadRunnerBridge\Centrifugo\ServiceInterface;

final readonly class ConnectService implements ServiceInterface
{
    public function __construct(
        private QueryBusInterface $bus,
    ) {}

    public function handle(RequestInterface $request): void
    {
        \assert($request instanceof Connect);

        /** @var Project[] $projects */
        $projects = $this->bus->ask(new FindAllProjects());

        /** @var non-empty-string[] $channels */
        $channels = [];
        foreach ($projects as $project) {
            $channels[] = (string) new EventsChannel($project->getKey());
        }

        try {
            $request->respond(
                new ConnectResponse(
                    user: (string) $request->getAttribute('user_id'),
                    channels: $channels,
                ),
            );
        } catch (\Throwable $e) {
            $request->error((int) $e->getCode(), $e->getMessage());
        }
    }
}
