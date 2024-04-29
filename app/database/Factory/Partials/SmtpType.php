<?php

declare(strict_types=1);

namespace Database\Factory\Partials;

trait SmtpType
{
    protected static function getSmtpPayload(): array
    {
        return [
            'id' => '30ec7d32c60314812bd8de2d26e8e751@local.host',
            'from' => [
                [
                    'email' => 'no-reply@site.com',
                    'name' => 'Bob Example',
                ],
            ],
            'reply_to' => [],
            'subject' => 'Test message',
            'to' => [
                [
                    'name' => 'Alice Example',
                    'email' => 'alice@example.com',
                ],
                [
                    'name' => '',
                    'email' => 'barney@example.com',
                ],
                [
                    'name' => 'John Doe',
                    'email' => 'john@example.com',
                ],
            ],
            'cc' => [
                [
                    'name' => '',
                    'email' => 'theboss@example.com',
                ],
                [
                    'name' => 'Customer',
                    'email' => 'customer@example.com',
                ],
            ],
            'bcc' => [],
            'text' => '',
            'html' => '',
            'raw' => "To: \"Alice Example\" <alice@example.com>, barney@example.com, \"John Doe\" <john@example.com>
Cc: theboss@example.com, \"Customer\" <customer@example.com>
Subject: Test message
From: \"Bob Example\" <no-reply@site.com>
Message-ID: <30ec7d32c60314812bd8de2d26e8e751@local.host>
MIME-Version: 1.0
Date: Sun, 28 Apr 2024 07:21:26 +0000
Content-Type: text/html; charset=utf-8
Content-Transfer-Encoding: quoted-printable<!doctype html><html><body>Hello Alice.
This is a test message with 5 header fields and 4 lines in the message body.</body></html>",
            'attachments' => [],
        ];
    }
}
