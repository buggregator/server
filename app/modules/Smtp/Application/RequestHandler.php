<?php

declare(strict_types=1);

namespace Modules\Smtp\Application;

use App\Application\Commands\HandleReceivedEvent;
use App\Application\Commands\StoreAttachment;
use App\Application\Domain\ValueObjects\Uuid;
use App\Application\Service\ClientProxy\EventHandlerInterface;
use Buggregator\Client\Proto\FilesCarrier;
use Buggregator\Client\Proto\Frame;
use Buggregator\Client\Traffic\Message\Smtp\Contact;
use Buggregator\Client\Traffic\Message\Smtp\MessageFormat;
use Modules\Attachments\Domain\Attachment;
use Spiral\Cqrs\CommandBusInterface;

final class RequestHandler implements EventHandlerInterface
{
    public function __construct(
        private readonly CommandBusInterface $commands,
    ) {
    }

    public function isSupported(Frame $frame): bool
    {
        return $frame instanceof Frame\Smtp;
    }

    /**
     * @param Frame\Smtp $frame
     */
    public function handle(Frame $frame): void
    {
        $payload = [
            'from' => $this->parseContacts($frame->message->getSender()),
            'reply_to' => $this->parseContacts($frame->message->getReplyTo()),
            'subject' => $frame->message->getSubject(),
            'to' => $this->parseContacts($frame->message->getTo()),
            'cc' => $this->parseContacts($frame->message->getCc()),
            'bcc' => $this->parseContacts($frame->message->getBcc()),
            'text' => $frame->message->getMessage(MessageFormat::Plain)?->getValue(),
            'html' => $frame->message->getMessage(MessageFormat::Html)?->getValue(),
            'raw' => (string)$frame->message->getBody(),
            'attachments' => [],
        ];

        $messageUuid = Uuid::generate();

        if ($frame instanceof FilesCarrier) {
            foreach ($frame->getFiles() as $file) {
                $uuid = Uuid::generate();

                /** @var Attachment $attachment */
                $attachment = $this->commands->dispatch(
                    new StoreAttachment(
                        uuid: $uuid,
                        parentUuid: $messageUuid,
                        file: $file,
                    )
                );

                $payload['attachments'][(string)$attachment->getUuid()] = [
                    'name' => $attachment->getFilename(),
                    'uri' => $attachment->getPath(),
                    'size' => $attachment->getSize(),
                    'mime' => $attachment->getMimeType(),
                    'id' => (string)$attachment->getUuid(),
                ];
            }
        };

        $this->commands->dispatch(
            new HandleReceivedEvent(type: 'smtp', payload: $payload, uuid: $messageUuid),
        );
    }

    private function parseContacts(array $contacts): array
    {
        return \array_map(
            static fn(Contact $contact): array => [
                'name' => $contact->name,
                'email' => $contact->email,
            ],
            $contacts,
        );
    }
}
