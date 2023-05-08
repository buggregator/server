<?php

declare(strict_types=1);

namespace Modules\Smtp\Application\Mail;

use JsonSerializable;
use Spiral\Storage\BucketInterface;

final class Message implements JsonSerializable
{
    private array $storedAttachments = [];

    /**
     * @param Attachment[] $attachments
     */
    public function __construct(
        private readonly BucketInterface $storage,
        private readonly ?string $id,
        private readonly string $raw,
        private readonly array $sender,
        private readonly array $recipients,
        private readonly array $ccs,
        private readonly string $subject,
        private readonly string $htmlBody,
        private readonly string $textBody,
        private readonly array $replyTo,
        private readonly array $allRecipients,
        private readonly array $attachments,
    ) {
    }

    /**
     * BCCs are recipients passed as RCPTs but not
     * in the body of the mail.
     *
     * @return non-empty-string[]
     */
    private function getBccs(): array
    {
        return \array_values(
            \array_filter($this->allRecipients, function (string $recipient) {
                foreach (\array_merge($this->recipients, $this->ccs) as $publicRecipient) {
                    if (\str_contains($publicRecipient, $recipient)) {
                        return false;
                    }
                }

                return true;
            }),
        );
    }

    public function jsonSerialize(): array
    {
        return [
            'id' => $this->id,
            'from' => $this->sender,
            'reply_to' => $this->replyTo,
            'subject' => $this->subject,
            'to' => $this->recipients,
            'cc' => $this->ccs,
            'bcc' => $this->getBccs(),
            'text' => $this->textBody,
            'html' => $this->htmlBody,
            'raw' => $this->raw,
            'attachments' => $this->storedAttachments,
        ];
    }

    public function storeAttachments(): self
    {
        foreach ($this->attachments as $attachment) {
            $file = $this->storage->write(
                $filename = $this->id . '/' . $attachment->getFilename(),
                $attachment->getContent(),
            );

            $this->storedAttachments[$attachment->getId()] = [
                'name' => $attachment->getFilename(),
                'uri' => $filename,
                'size' => $file->getSize(),
                'mime' => $file->getMimeType(),
                'id' => $attachment->getId(),
            ];
        }

        return $this;
    }
}
