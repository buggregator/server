<?php

declare(strict_types=1);

namespace Modules\Smtp\Interfaces\Jobs;

use App\Application\Commands\HandleReceivedEvent;
use App\Application\Domain\ValueObjects\Uuid;
use Modules\Smtp\Application\Mail\Attachment;
use Modules\Smtp\Application\Mail\Message;
use Modules\Smtp\Domain\AttachmentStorageInterface;
use Spiral\Core\InvokerInterface;
use Spiral\Cqrs\CommandBusInterface;
use Spiral\Queue\JobHandler;
use ZBateson\MailMimeParser\Message as ParseMessage;

final class EmailHandler extends JobHandler
{
    public function __construct(
        private readonly CommandBusInterface $bus,
        private readonly AttachmentStorageInterface $attachments,
        InvokerInterface $invoker,
    ) {
        parent::__construct($invoker);
    }

    public function invoke(mixed $payload): void
    {
        $data = $payload;
        $rawBody = $data['message']['raw'];

        $message = ParseMessage::from($rawBody, true);

        $this->dispatchMessage(new Message(
            id: $data['message']['id'],
            raw: $rawBody,
            sender: $data['envelope']['from'],
            recipients: $data['envelope']['to'],
            ccs: $data['envelope']['ccs'],
            subject: $message->getHeaderValue('subject', $data['message']['subject']),
            htmlBody: (string) $message->getHtmlContent(),
            textBody: (string) $message->getTextContent(),
            replyTo: $data['envelope']['replyTo'],
            allRecipients: $data['envelope']['allRecipients'],
            attachments: \array_map(
                static fn(array $attachment) => new Attachment(
                    filename: $attachment['filename'],
                    content: \base64_decode($attachment['content'] ?? ''),
                    type: $attachment['type'] ?? 'application/octet-stream',
                    contentId: $attachment['contentId'] ?? null,
                ),
                $data['attachments'],
            ),
        ));
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
}
