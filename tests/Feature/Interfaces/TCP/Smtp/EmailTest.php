<?php

declare(strict_types=1);

namespace Tests\Feature\Interfaces\TCP\Smtp;

use App\Application\Broadcasting\Channel\EventsChannel;
use Modules\Smtp\Application\Storage\EmailBodyStorage;
use Modules\Smtp\Application\Storage\Message;
use Modules\Smtp\Interfaces\TCP\Service as SmtpService;
use Ramsey\Uuid\Uuid;
use Spiral\RoadRunner\Tcp\TcpEvent;
use Spiral\RoadRunnerBridge\Tcp\Response\CloseConnection;
use Symfony\Component\Mailer\Transport\Smtp\EsmtpTransport;
use Symfony\Component\Mime\Address;
use Symfony\Component\Mime\Email;
use Tests\App\Smtp\FakeStream;
use Tests\Feature\Interfaces\TCP\TCPTestCase;

final class EmailTest extends TCPTestCase
{
    public function testSendEmail(): void
    {
        $this->createProject('default');

        $email = (new Email)
            ->subject('Test message')
            ->date(new \DateTimeImmutable('2024-05-02 16:01:33'))
            ->addTo(
                new Address('alice@example.com', 'Alice Doe'),
                'barney@example.com',
                new Address('john@example.com', 'John Doe'),
            )
            ->addCc(
                new Address('customer@example.com', 'Customer'),
                'theboss@example.com',
            )
            ->addFrom(
                new Address('no-reply@site.com', 'Bob Example'),
            )
            ->text(
                body: 'Hello Alice.<br>This is a test message with 5 header fields and 4 lines in the message body.'
            );

        $email->getHeaders()->addIdHeader('Message-ID', $email->generateMessageId());
        $id = $email->getHeaders()->get('Message-ID')->getBody()[0];

        $client = new EsmtpTransport(
            stream: new FakeStream(
                service: $this->get(SmtpService::class),
                uuid: $uuid = Uuid::uuid7()->toString(),
            ),
        );

        $client->setUsername('default');
        $client->setPassword('password');

        $client->send($email);

        $this->validateMessage($id, $uuid);

        $response = $this->handleSmtpRequest(message: '', event: TCPEvent::Close);
        $this->assertInstanceOf(CloseConnection::class, $response);

        $this->assertEventPushed('default');
    }

    private function getEmailMessage(string $uuid): Message
    {
        return $this->get(EmailBodyStorage::class)->getMessage($uuid);
    }

    private function validateMessage(string $messageId, string $uuid): void
    {
        $messageData = $this->getEmailMessage($uuid);

        $this->assertSame('default', $messageData->username);
        $this->assertSame('password', $messageData->password);
        $this->assertSame('no-reply@site.com', $messageData->from);
        $this->assertSame([
            'alice@example.com',
            'barney@example.com',
            'john@example.com',
            'customer@example.com',
            'theboss@example.com',
        ], $messageData->recipients);

        $this->assertSame([
            'id' => $messageId,
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
                    'name' => 'Alice Doe',
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
                    'name' => 'Customer',
                    'email' => 'customer@example.com',
                ],
                [
                    'name' => '',
                    'email' => 'theboss@example.com',
                ],
            ],
            'bcc' => [],
            'text' => 'Hello Alice.<br>This is a test message with 5 header fields and 4 lines in the message body.',
            'html' => '',
            'raw' => "Subject: Test message\r
Date: Thu, 02 May 2024 16:01:33 +0000\r
To: Alice Doe <alice@example.com>, barney@example.com, John Doe\r
 <john@example.com>\r
Cc: Customer <customer@example.com>, theboss@example.com\r
From: Bob Example <no-reply@site.com>\r
Message-ID: <$messageId>\r
MIME-Version: 1.0\r
Content-Type: text/plain; charset=utf-8\r
Content-Transfer-Encoding: quoted-printable\r
\r
Hello Alice.<br>This is a test message with 5 header fields and 4 lines in =\r
the message body.",
        ], $messageData->parse()->jsonSerialize());
    }

    private function assertEventPushed(?string $project = null): void
    {
        $this->broadcastig->assertPushed(new EventsChannel($project), function (array $data) use ($project) {
            $this->assertSame('event.received', $data['event']);
            $this->assertSame('smtp', $data['data']['type']);
            $this->assertSame($project, $data['data']['project']);

            $this->assertSame([
                'subject' => 'Test message',
                'from' => [
                    [
                        'email' => 'no-reply@site.com',
                        'name' => 'Bob Example',
                    ],
                ],
                'to' => [
                    [
                        'name' => 'Alice Doe',
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
            ], $data['data']['payload']);

            $this->assertNotEmpty($data['data']['uuid']);
            $this->assertNotEmpty($data['data']['timestamp']);

            return true;
        });
    }
}
