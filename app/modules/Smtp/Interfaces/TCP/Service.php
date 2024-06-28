<?php

declare(strict_types=1);

namespace Modules\Smtp\Interfaces\TCP;

use App\Application\Commands\HandleReceivedEvent;
use App\Application\Domain\ValueObjects\Uuid;
use Modules\Smtp\Application\Mail\Message;
use Modules\Smtp\Application\Storage\EmailBodyStorage;
use Modules\Smtp\Domain\AttachmentStorageInterface;
use Spiral\Cqrs\CommandBusInterface;
use Spiral\RoadRunner\Tcp\Request;
use Spiral\RoadRunner\Tcp\TcpEvent;
use Spiral\RoadRunner\Tcp\TcpResponse;
use Spiral\RoadRunnerBridge\Tcp\Response\CloseConnection;
use Spiral\RoadRunnerBridge\Tcp\Response\RespondMessage;
use Spiral\RoadRunnerBridge\Tcp\Response\ResponseInterface;
use Spiral\RoadRunnerBridge\Tcp\Service\ServiceInterface;

final readonly class Service implements ServiceInterface
{
    public function __construct(
        private CommandBusInterface $bus,
        private EmailBodyStorage $emailBodyStorage,
        private AttachmentStorageInterface $attachments,
    ) {}

    public function handle(Request $request): ResponseInterface
    {
        if ($request->event === TcpEvent::Connected) {
            return $this->send(ResponseMessage::ready());
        }

        $message = $this->emailBodyStorage->getMessage($request->connectionUuid);

        $response = new CloseConnection();
        $dispatched = false;

        if ($request->event === TcpEvent::Close) {
            $this->emailBodyStorage->delete($message);

            return new CloseConnection();
        } elseif (\preg_match('/^(EHLO|HELO)/', $request->body)) {
            $response = $this->sendMultiply(
                ResponseMessage::ok(separator: '-buggregator'),
                ResponseMessage::authRequired(),
            );
        } elseif (\preg_match('/^MAIL FROM:\s*<(.*)>/', $request->body, $matches)) {
            $message->setFrom($matches[1]);
            $response = $this->send(ResponseMessage::ok());
        } elseif (\str_starts_with($request->body, 'AUTH')) {
            $response = $this->send(ResponseMessage::enterUsername());
            $message->waitUsername = true;
        } elseif ($message->waitUsername) {
            $message->setUsername($request->body);
            $response = $this->send(ResponseMessage::enterPassword());
        } elseif ($message->waitPassword) {
            $message->setPassword($request->body);
            $response = $this->send(ResponseMessage::authenticated());
        } elseif (\preg_match('/^RCPT TO:\s*<(.*)>/', $request->body, $matches)) {
            $message->addRecipient($matches[1]);
            $response = $this->send(ResponseMessage::ok());
        } elseif (\str_starts_with($request->body, 'QUIT')) {
            $response = $this->send(ResponseMessage::closing(), close: true);
        } elseif ($request->body === "DATA\r\n") {
            $response = $this->send(ResponseMessage::provideBody());
            $message->waitBody = true;
        } elseif ($message->waitBody) {
            $response = $this->send(ResponseMessage::ok());
            $message->appendBody($request->body);

            if ($message->bodyHasEos()) {
                $this->dispatchMessage($message->parse(), project: $message->username);
                $dispatched = true;
            }
        }

        if (
            $response instanceof CloseConnection ||
            $response->getAction() === TcpResponse::RespondClose ||
            $dispatched
        ) {
            return $response;
        }

        $this->emailBodyStorage->persist($message);

        return $response;
    }

    private function dispatchMessage(Message $message, ?string $project = null): void
    {
        $uuid = Uuid::generate();
        $data = $message->jsonSerialize();


        $result = $this->attachments->store(eventUuid: $uuid, attachments: $message->attachments);
        // TODO: Refactor this
        foreach ($result as $cid => $url) {
            $data['html'] = \str_replace('cid:' . $cid, $url, $data['html']);
        }

        $this->bus->dispatch(
            new HandleReceivedEvent(
                type: 'smtp',
                payload: $data,
                project: $project,
                uuid: $uuid,
            ),
        );
    }

    private function send(ResponseMessage $message, bool $close = false): RespondMessage
    {
        return new RespondMessage((string) $message, $close);
    }

    private function sendMultiply(ResponseMessage...$message): RespondMessage
    {
        return new RespondMessage(\implode("", $message));
    }
}
