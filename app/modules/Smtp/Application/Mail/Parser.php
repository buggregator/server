<?php

declare(strict_types=1);

namespace Modules\Smtp\Application\Mail;

use ZBateson\MailMimeParser\Header\AbstractHeader;
use ZBateson\MailMimeParser\Header\AddressHeader;
use ZBateson\MailMimeParser\Header\Part\AddressPart;
use ZBateson\MailMimeParser\Message as ParseMessage;

final readonly class Parser
{
    public function parse(string $body, array $allRecipients = []): Message
    {
        $message = ParseMessage::from($body, true);

        /** @var AddressPart $fromData */
        $fromData = $message->getHeader('from')?->getParts()[0] ?? null;
        $from = [['email' => $fromData?->getValue(), 'name' => $fromData?->getName()]];

        /** @var AddressHeader|null $toHeader */
        $toHeader = $message->getHeader('to');
        $recipients = $this->joinNameAndEmail($toHeader ? $toHeader->getAddresses() : []);
        /** @var AddressHeader|null $ccHeader */
        $ccHeader = $message->getHeader('cc');
        $ccs = $this->joinNameAndEmail($ccHeader ? $ccHeader->getAddresses() : []);
        $subject = (string)$message->getHeaderValue('subject');
        $html = (string)$message->getHtmlContent();
        $text = (string)$message->getTextContent();
        /** @var AbstractHeader|null $replyToHeader */
        $replyToHeader = $message->getHeader('reply-to')?->getParts()[0] ?? null;
        $replyTo = $replyToHeader ? [
            [
                'email' => $replyToHeader?->getValue(),
                'name' => $replyToHeader?->getName(),
            ],
        ] : [];

        $attachments = $this->buildAttachmentFrom(
            $message->getAllAttachmentParts(),
        );

        return new Message(
            $message->getHeader('Message - Id')?->getValue(),
            $body, $from, $recipients, $ccs, $subject,
            $html, $text, $replyTo, $allRecipients, $attachments,
        );
    }

    /**
     * @param ParseMessage\IMessagePart[] $attachments
     * @return Attachment[]
     */
    private function buildAttachmentFrom(array $attachments): array
    {
        return \array_map(fn(ParseMessage\IMessagePart $part) => new Attachment(
            $part->getFilename(),
            $part->getContent(),
            $part->getContentType(),
        ), $attachments);
    }

    /**
     * @param AddressPart[] $addresses
     * @return string[]
     */
    private function joinNameAndEmail(array $addresses): array
    {
        return \array_map(function (AddressPart $addressPart) {
            $name = $addressPart->getName();
            $email = $addressPart->getValue();

            return ['name' => $name, 'email' => $email];
        }, $addresses);
    }
}
