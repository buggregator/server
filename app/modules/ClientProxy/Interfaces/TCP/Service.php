<?php

declare(strict_types=1);

namespace Modules\ClientProxy\Interfaces\TCP;

use App\Application\Commands\HandleReceivedEvent;
use Spiral\Cqrs\CommandBusInterface;
use Spiral\RoadRunner\Tcp\Request;
use Spiral\RoadRunner\Tcp\TcpWorkerInterface;
use Spiral\RoadRunnerBridge\Tcp\Response\ContinueRead;
use Spiral\RoadRunnerBridge\Tcp\Response\ResponseInterface;
use Spiral\RoadRunnerBridge\Tcp\Service\ServiceInterface;

class Service implements ServiceInterface
{
    public function __construct(
        private readonly CommandBusInterface $commandBus,
    ) {
    }

    public function handle(Request $request): ResponseInterface
    {
        dump($request);
        if ($request->event === TcpWorkerInterface::EVENT_CONNECTED) {
            return new ContinueRead();
        }

        $messages = \array_filter(\explode("\n", $request->body));

        foreach ($messages as $message) {
            //dump($message);
        }

        return new ContinueRead();
    }

}
