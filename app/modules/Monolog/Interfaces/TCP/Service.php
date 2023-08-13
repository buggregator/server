<?php

declare(strict_types=1);

namespace Modules\Monolog\Interfaces\TCP;

use Modules\Monolog\Application\RequestHandler;
use Spiral\RoadRunner\Tcp\Request;
use Spiral\RoadRunner\Tcp\TcpWorkerInterface;
use Spiral\RoadRunnerBridge\Tcp\Response\CloseConnection;
use Spiral\RoadRunnerBridge\Tcp\Response\ContinueRead;
use Spiral\RoadRunnerBridge\Tcp\Response\ResponseInterface;
use Spiral\RoadRunnerBridge\Tcp\Service\ServiceInterface;

class Service implements ServiceInterface
{
    public function __construct(
        private readonly RequestHandler $requestHandler,
    ) {
    }

    public function handle(Request $request): ResponseInterface
    {
        if ($request->event === TcpWorkerInterface::EVENT_CONNECTED) {
            return new ContinueRead();
        }

        $messages = \array_filter(\explode("\n", $request->body));

        foreach ($messages as $message) {
            $this->requestHandler->handle($message);
        }

        return new CloseConnection();
    }
}
