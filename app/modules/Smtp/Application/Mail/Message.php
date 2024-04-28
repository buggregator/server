<?php

declare(strict_types=1);

namespace Modules\Smtp\Application\Mail;

use JsonSerializable;

final readonly class Message implements JsonSerializable
{
    /**
     * @param Attachment[] $attachments
     */
    public function __construct(
        public ?string $id,
        public string $raw,
        public array $sender,
        public array $recipients,
        public array $ccs,
        public string $subject,
        public string $htmlBody,
        public string $textBody,
        public array $replyTo,
        public array $allRecipients,
        public array $attachments,
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
        ];
    }

    public function storeAttachments(): self
    {


        return $this;
    }
}
