<?php

declare(strict_types=1);

namespace Trap;

use App\Application\Service\ClientProxy\EventHandlerRegistryInterface;
use Tests\TestCase;

final class SmtpTest extends TestCase
{
    public function testHandleSmtpFrame(): void
    {
        $body = \str_split(
            <<<SMTP
            From: sender@example.com\r
            To: recipient@example.com\r
            Subject: Multipart Email Example\r
            Content-Type: multipart/alternative; boundary="boundary-string"\r
            \r
            --boundary-string\r
            Content-Type: text/plain; charset="utf-8"\r
            Content-Transfer-Encoding: quoted-printable\r
            Content-Disposition: inline\r
            \r
            Plain text email goes here!\r
            This is the fallback if email client does not support HTML\r
            \r
            --boundary-string\r
            Content-Type: text/html; charset="utf-8"\r
            Content-Transfer-Encoding: quoted-printable\r
            Content-Disposition: inline\r
            \r
            <h1>This is the HTML Section!</h1>\r
            <p>This is what displays in most modern email clients</p>\r
            \r
            --boundary-string--\r
            Content-Type: image/x-icon\r
            Content-Transfer-Encoding: base64\r
            Content-Disposition: attachment;filename=logo.ico\r
            \r
            123456789098765432123456789\r
            \r
            --boundary-string--\r
            Content-Type: text/watch-html; charset="utf-8"\r
            Content-Transfer-Encoding: quoted-printable\r
            Content-Disposition: inline\r
            \r
            <b>Apple Watch formatted content</b>\r
            \r
            --boundary-string--\r\n\r\n
            SMTP,
            10
        );

        $frame = $this->createSmtpFrame($body);

        /** @var EventHandlerRegistryInterface $handler */
        $handler = $this->getContainer()->get(EventHandlerRegistryInterface::class);

        $handler->handle($frame);
    }
}
