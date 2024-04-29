<?php

declare(strict_types=1);

namespace App\Interfaces\Centrifugo;

use App\Application\Broadcasting\Channel\EventsChannel;
use App\Application\Commands\FindAllProjects;
use Modules\Projects\Domain\Project;
use RoadRunner\Centrifugo\Payload\ConnectResponse;
use RoadRunner\Centrifugo\Request;
use RoadRunner\Centrifugo\Request\RequestInterface;
use Spiral\Cqrs\QueryBusInterface;
use Spiral\RoadRunnerBridge\Centrifugo\ServiceInterface;

final readonly class ConnectService implements ServiceInterface
{
    public function __construct(
        private QueryBusInterface $bus,
    ) {
    }

    /**
     * @param Request\Connect $request
     */
    public function handle(RequestInterface $request): void
    {
        /** @var Project[] $projects */
        $projects = $this->bus->ask(new FindAllProjects());
        $channels = [new EventsChannel()];

        foreach ($projects as $project) {
            $channels[] = new EventsChannel($project->getKey());
        }

        try {
            $request->respond(
                new ConnectResponse(
                    user: (string)$request->getAttribute('user_id'),
                    channels: \array_map(
                        static fn(EventsChannel $channel) => (string)$channel,
                        $channels,
                    ),
                ),
            );
        } catch (\Throwable $e) {
            $request->error($e->getCode(), $e->getMessage());
        }
    }
}
