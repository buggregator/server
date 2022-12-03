<?php

declare(strict_types=1);

namespace Modules\VarDumper\Interfaces\TCP;

use App\Application\Commands\HandleReceivedEvent;
use Modules\VarDumper\Application\Dump\MessageParser;
use Psr\Log\LoggerInterface;
use Spiral\Cqrs\CommandBusInterface;
use Spiral\RoadRunner\Tcp\Request;
use Spiral\RoadRunner\Tcp\TcpWorkerInterface;
use Spiral\RoadRunnerBridge\Tcp\Response\CloseConnection;
use Spiral\RoadRunnerBridge\Tcp\Response\ContinueRead;
use Spiral\RoadRunnerBridge\Tcp\Response\ResponseInterface;
use Spiral\RoadRunnerBridge\Tcp\Service\ServiceInterface;
use Symfony\Component\Console\Input\ArrayInput;
use Symfony\Component\Console\Output\OutputInterface;
use Symfony\Component\Console\Style\SymfonyStyle;
use Symfony\Component\VarDumper\Cloner\Data;
use Symfony\Component\VarDumper\Command\Descriptor\CliDescriptor;
use Symfony\Component\VarDumper\Dumper\CliDumper;
use Symfony\Component\VarDumper\Dumper\HtmlDumper;

class Service implements ServiceInterface
{
    public function __construct(
        private readonly CommandBusInterface $commandBus,
        private readonly LoggerInterface $logger
    ) {
    }

    public function handle(Request $request): ResponseInterface
    {
        if ($request->event === TcpWorkerInterface::EVENT_CONNECTED) {
            return new ContinueRead();
        }

        $messages = \array_filter(\explode("\n", $request->body));

        foreach ($messages as $message) {
            $payload = (new MessageParser())->parse($message);
            $this->fireEvent($payload);
        }

        return new CloseConnection();
    }

    private function fireEvent(array $payload): void
    {
        $this->commandBus->dispatch(
            new HandleReceivedEvent(
                type: 'var-dump',
                payload: [
                    'payload' => [
                        'type' => $payload[0]->getType(),
                        'value' => $this->convertToPrimitive($payload[0]),
                    ],
                    'context' => $payload[1],
                ]
            )
        );
    }

    private function convertToPrimitive(Data $data): string|null
    {
        if (\in_array($data->getType(), ['string', 'boolean'])) {
            return (string)$data->getValue();
        }

        $dumper = new HtmlDumper();

        return $dumper->dump($data, true);
    }

    private function sendToConsole(Request $request, array $payload, OutputInterface $output): void
    {
        $descriptor = new CliDescriptor(new CliDumper());

        [$data, $context] = $payload;

        $context['cli']['identifier'] = $request->connectionUuid;
        $context['cli']['command_line'] = $request->remoteAddr;

        $descriptor->describe(new SymfonyStyle(new ArrayInput([]), $output), $data, $context, 0);
    }
}
