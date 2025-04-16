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
            return $this->makeResponse(ResponseMessage::ready());
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
            $response = $this->makeResponse(ResponseMessage::ok());
        } elseif (\str_starts_with($request->body, 'AUTH')) {
            $response = $this->makeResponse(ResponseMessage::enterUsername());
            $message->waitUsername = true;
        } elseif ($message->waitUsername) {
            $message->setUsername($request->body);
            $response = $this->makeResponse(ResponseMessage::enterPassword());
        } elseif ($message->waitPassword) {
            $message->setPassword($request->body);
            $response = $this->makeResponse(ResponseMessage::authenticated());
        } elseif (\preg_match('/^RCPT TO:\s*<(.*)>/', $request->body, $matches)) {
            $message->addRecipient($matches[1]);
            $response = $this->makeResponse(ResponseMessage::ok());
        } elseif (\str_starts_with($request->body, 'QUIT')) {
            $response = $this->makeResponse(ResponseMessage::closing(), close: true);
            $message = $this->emailBodyStorage->cleanup($request->connectionUuid);
        } elseif ($request->body === "DATA\r\n") {
            // Reset the body to empty string when starting a new DATA command
            // This prevents confusion between multiple DATA commands in the same session
            $message->body = '';
            $response = $this->makeResponse(ResponseMessage::provideBody());
            $message->waitBody = true;
        } elseif ($request->body === "RSET\r\n") {
            $message = $this->emailBodyStorage->cleanup($request->connectionUuid);
            $response = $this->makeResponse(ResponseMessage::ok());
        } elseif ($request->body === "NOOP\r\n") {
            $response = $this->makeResponse(ResponseMessage::ok());
        } elseif ($message->waitBody) {
            $message->appendBody($request->body);

            // FIX: Only send one response when data ends
            if ($message->bodyHasEos()) {
                $uuid = $this->dispatchMessage($message->parse(), project: $message->username);
                $response = $this->makeResponse(ResponseMessage::accepted($uuid));
                $dispatched = true;
                // Reset the waitBody flag to false since we've processed the message
                $message->waitBody = false;
            } else {
                // Only send "OK" response if we're not at the end of data
                $response = $this->makeResponse(ResponseMessage::ok());
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

    private function dispatchMessage(Message $message, ?string $project = null): Uuid
    {
        $uuid = Uuid::generate();
        $data = $message->jsonSerialize();

        $result = $this->attachments->store(eventUuid: $uuid, attachments: $message->attachments);
        // TODO: Refactor this
        foreach ($result as $cid => $url) {
            $data['html'] = \str_replace("cid:$cid", $url, $data['html']);
        }

        $this->bus->dispatch(
            new HandleReceivedEvent(
                type: 'smtp',
                payload: $data,
                project: $project,
                uuid: $uuid,
            ),
        );

        return $uuid;
    }

    private function makeResponse(ResponseMessage $message, bool $close = false): RespondMessage
    {
        return new RespondMessage((string) $message, $close);
    }

    private function sendMultiply(ResponseMessage...$message): RespondMessage
    {
        return new RespondMessage(\implode("", $message));
    }
}
