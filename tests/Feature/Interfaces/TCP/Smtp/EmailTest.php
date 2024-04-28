<?php

declare(strict_types=1);

namespace Tests\Feature\Interfaces\TCP\Smtp;

use App\Application\Broadcasting\Channel\EventsChannel;
use Modules\Smtp\Application\Storage\EmailBodyStorage;
use Modules\Smtp\Application\Storage\Message;
use Modules\Smtp\Interfaces\TCP\ResponseMessage;
use Spiral\RoadRunner\Tcp\TcpEvent;
use Spiral\RoadRunnerBridge\Tcp\Response\CloseConnection;
use Tests\Feature\Interfaces\TCP\TCPTestCase;

final class EmailTest extends TCPTestCase
{
    public function testSendEmail(): void
    {
        $this->createProject('default');

        $body = <<<'BODY'
To: "Alice Example" <alice@example.com>, barney@example.com, "John Doe" <john@example.com>
Cc: theboss@example.com, "Customer" <customer@example.com>
Subject: Test message
From: "Bob Example" <no-reply@site.com>
Message-ID: <30ec7d32c60314812bd8de2d26e8e751@local.host>
MIME-Version: 1.0
Date: Sun, 28 Apr 2024 07:21:26 +0000
Content-Type: text/html; charset=utf-8
Content-Transfer-Encoding: quoted-printable
BODY;

        $response = $this->handleSmtpRequest(message: 'EHLO site.com', event: TcpEvent::Connected);
        $this->assertSame((string)ResponseMessage::ready(), $response->getBody());

        $flow = [
            'HELO site.com' => ResponseMessage::ok(),
            'MAIL FROM:<no-reply@site.com>' => ResponseMessage::ok(),
            'RCPT TO:<alice@example.com>' => ResponseMessage::ok(),
            'RCPT TO:<barney@example.com>' => ResponseMessage::ok(),
            'RCPT TO:<john@example.com>' => ResponseMessage::ok(),
            // 250 AUTH LOGIN PLAIN CRAM-MD5 \r\n
            "AUTH LOGIN\r\n" => ResponseMessage::enterUsername(),
            \base64_encode('default') . "\r\n" => ResponseMessage::enterPassword(),
            \base64_encode('password') . "\r\n" => ResponseMessage::authenticated(),
            "DATA\r\n" => ResponseMessage::provideBody(),
            $body => ResponseMessage::ok(),
            "<!doctype html><html><body>Hello Alice.\nThis is a test message with 5 header fields and 4 lines in the message body.</body></html>\r\n.\r\n" => ResponseMessage::ok(
            ),
        ];

        foreach ($flow as $message => $resp) {
            $response = $this->handleSmtpRequest(message: $message);
            $this->assertSame((string)$resp, $response->getBody());
        }

        $this->validateMessage();

        $response = $this->handleSmtpRequest(message: 'QUIT');
        $this->assertSame((string)ResponseMessage::closing(), $response->getBody());

        $response = $this->handleSmtpRequest(message: '', event: TCPEvent::Close);
        $this->assertInstanceOf(CloseConnection::class, $response);

        $this->assertEventPushed();
    }

    private function getEmailMessage(string $uuid): Message
    {
        return $this->get(EmailBodyStorage::class)->getMessage($uuid);
    }

    private function validateMessage(): void
    {
        $messageData = $this->getEmailMessage('018f2586-4be9-7168-942e-0ce0c104961');
        $this->assertSame('default', $messageData->username);
        $this->assertSame('password', $messageData->password);
        $this->assertSame('no-reply@site.com', $messageData->from);
        $this->assertSame(['alice@example.com', 'barney@example.com', 'john@example.com'], $messageData->recipients);
    }

    private function assertEventPushed(?string $project = null): void
    {
        $this->broadcastig->assertPushed(new EventsChannel($project), function (array $data) use ($project) {
            $this->assertSame('event.received', $data['event']);
            $this->assertSame('smtp', $data['data']['type']);
            $this->assertSame($project, $data['data']['project']);

            $this->assertSame([
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
            ], $data['data']['payload']);

            $this->assertNotEmpty($data['data']['uuid']);
            $this->assertNotEmpty($data['data']['timestamp']);

            return true;
        });
    }
}
