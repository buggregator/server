<?php

declare(strict_types=1);

namespace Tests\Feature\Interfaces\TCP\Smtp;

use Spiral\Storage\BucketInterface;
use Mockery\MockInterface;
use App\Application\Broadcasting\Channel\EventsChannel;
use Modules\Smtp\Application\Storage\EmailBodyStorage;
use Modules\Smtp\Application\Storage\Message;
use Modules\Smtp\Domain\Attachment;
use Modules\Smtp\Domain\AttachmentRepositoryInterface;
use Ramsey\Uuid\Uuid;
use Spiral\RoadRunner\Tcp\TcpEvent;
use Spiral\RoadRunnerBridge\Tcp\Response\CloseConnection;
use Symfony\Component\Mailer\SentMessage;
use Symfony\Component\Mime\Address;
use Symfony\Component\Mime\Email;
use Symfony\Component\Mime\Part\DataPart;
use Symfony\Component\Mime\Part\File;
use Tests\Feature\Interfaces\TCP\TCPTestCase;

final class EmailTest extends TCPTestCase
{
    private BucketInterface $bucket;
    private MockInterface|AttachmentRepositoryInterface $attachments;

    protected function setUp(): void
    {
        parent::setUp();

        $this->bucket = $this->fakeStorage()->bucket('smtp');
        $this->attachments = $this->mockContainer(AttachmentRepositoryInterface::class);
    }

    public function testSendEmail(): void
    {
        $project = $this->createProject('foo');

        $email = $this->buildEmail();
        $email->getHeaders()->addIdHeader('Message-ID', $id = $email->generateMessageId());

        $client = $this->buildSmtpClient(
            username: (string) $project->getKey(),
            uuid: $connectionUuid = Uuid::uuid7(),
        );

        // Assert logo-embeddable is persisted to a database
        $this->attachments->shouldReceive('store')
            ->once()
            ->with(
                \Mockery::on(function (Attachment $attachment) {
                    $this->assertSame('logo-embeddable', $attachment->getFilename());
                    $this->assertSame(1207, $attachment->getSize());
                    $this->assertSame('image/svg+xml', $attachment->getMime());

                    // Check attachments storage
                    $this->bucket->assertCreated($attachment->getPath());
                    return true;
                }),
            );

        // Assert hello.txt is persisted to a database
        $this->attachments->shouldReceive('store')
            ->once()
            ->with(
                \Mockery::on(function (Attachment $attachment) {
                    $this->assertSame('hello.txt', $attachment->getFilename());
                    $this->assertSame(13, $attachment->getSize());
                    $this->assertSame('text/plain', $attachment->getMime());

                    // Check attachments storage
                    $this->bucket->assertCreated($attachment->getPath());
                    return true;
                }),
            );

        // Assert sample.pdf is persisted to a database
        $this->attachments->shouldReceive('store')
            ->once()
            ->with(
                \Mockery::on(function (Attachment $attachment) {
                    $this->assertSame('sample.pdf', $attachment->getFilename());
                    $this->assertSame(61752, $attachment->getSize());
                    $this->assertSame('application/pdf', $attachment->getMime());

                    // Check attachments storage
                    $this->bucket->assertCreated($attachment->getPath());
                    return true;
                }),
            );

        // Assert logo.svg is persisted to a database
        $this->attachments->shouldReceive('store')
            ->once()
            ->with(
                \Mockery::on(function (Attachment $attachment) {
                    $this->assertSame('logo.svg', $attachment->getFilename());
                    $this->assertSame(1207, $attachment->getSize());
                    $this->assertSame('image/svg+xml', $attachment->getMime());

                    // Check attachments storage
                    $this->bucket->assertCreated($attachment->getPath());
                    return true;
                }),
            );

        $sentMessage = $client->send($email);

        $this->validateMessage($id, (string) $connectionUuid);

        $response = $this->handleSmtpRequest(message: '', event: TCPEvent::Close);
        $this->assertInstanceOf(CloseConnection::class, $response);

        $this->assertEventPushed($sentMessage, 'foo');
    }

    public function testSendMultipleEmails(): void
    {
        $project = $this->createProject('foo');
        $connectionUuid = Uuid::uuid7();

        // We'll send two emails in the same SMTP session
        $email1 = $this->buildEmail();
        $email1->getHeaders()->addIdHeader('Message-ID', $id1 = $email1->generateMessageId());

        $email2 = $this->buildEmailWithCyrillic();
        $email2->getHeaders()->addIdHeader('Message-ID', $id2 = $email2->generateMessageId());

        // Set up attachment expectations (fixed to 7 based on the actual number of attachments)
        // The first email has 4 attachments, the second has 3
        $this->attachments->shouldReceive('store')->times(7)->andReturn(true);

        // Build SMTP client
        $client = $this->buildSmtpClient(
            username: (string) $project->getKey(),
            uuid: $connectionUuid,
        );

        // Send first email
        $sentMessage1 = $client->send($email1);

        // Validate the first message
        $this->validateMessage($id1, (string) $connectionUuid);
        $this->assertEventPushed($sentMessage1, 'foo');

        // Check that state is reset properly by sending second email
        $sentMessage2 = $client->send($email2);

        // This would fail before our fix if the state wasn't properly reset
        $messageData2 = $this->getEmailMessage((string) $connectionUuid);
        $this->assertFalse($messageData2->waitBody, 'waitBody flag should be reset after sending email');

        $response = $this->handleSmtpRequest(message: '', event: TCPEvent::Close);
        $this->assertInstanceOf(CloseConnection::class, $response);
    }

    private function getEmailMessage(string $uuid): Message
    {
        return $this->get(EmailBodyStorage::class)->getMessage($uuid);
    }

    private function validateMessage(string $messageId, string $uuid): void
    {
        $messageData = $this->getEmailMessage($uuid);

        $this->assertSame('foo', $messageData->username);
        $this->assertSame('password', $messageData->password);
        $this->assertSame('no-reply@site.com', $messageData->from);
        $this->assertSame([
            'alice@example.com',
            'barney@example.com',
            'john@example.com',
            'customer@example.com',
            'theboss@example.com',
        ], $messageData->recipients);

        $parsedMessage = $messageData->parse();

        $this->assertSame($messageId, $parsedMessage->id);
        $this->assertSame(
            [
                [
                    'email' => 'no-reply@site.com',
                    'name' => 'Bob Example',
                ],
            ],
            $parsedMessage->sender,
        );
        $this->assertSame([], $parsedMessage->replyTo);
        $this->assertSame('Test message', $parsedMessage->subject);
        $this->assertSame([
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
        ], $parsedMessage->recipients);
        $this->assertSame([
            [
                'name' => 'Customer',
                'email' => 'customer@example.com',
            ],
            [
                'name' => '',
                'email' => 'theboss@example.com',
            ],
        ], $parsedMessage->ccs);

        $this->assertSame([], $parsedMessage->getBccs());

        $this->assertStringEqualsStringIgnoringLineEndings(
            <<<'HTML'
<img src="cid:test-cid@buggregator">
Hello Alice.<br>This is a test message with 5 header fields and 4 lines in the message body.
HTML
            ,
            $parsedMessage->htmlBody,
        );

        $this->assertStringContainsString(
            "Subject: Test message\r
Date: Thu, 02 May 2024 16:01:33 +0000\r
To: Alice Doe <alice@example.com>, barney@example.com, John Doe\r
 <john@example.com>\r
Cc: Customer <customer@example.com>, theboss@example.com\r
From: Bob Example <no-reply@site.com>\r
Message-ID: <$messageId>\r",
            $parsedMessage->raw,
        );
    }

    private function assertEventPushed(SentMessage $message, ?string $project = null): void
    {
        $this->broadcastig->assertPushed(new EventsChannel($project), function (array $data) use ($message, $project) {
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

            $this->assertSame($message->getMessageId(), $data['data']['uuid']);
            $this->assertNotEmpty($data['data']['timestamp']);

            return true;
        });
    }

    public function buildEmail(): Email
    {
        return (new Email())
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
            ->addFrom(new Address('no-reply@site.com', 'Bob Example'), )
            ->attachFromPath(path: __DIR__ . '/hello.txt', )
            ->attachFromPath(path: __DIR__ . '/sample.pdf', )
            ->attachFromPath(path: __DIR__ . '/logo.svg')
            ->addPart(
                (new DataPart(new File(__DIR__ . '/logo.svg'), 'logo-embeddable'))->asInline()->setContentId(
                    'test-cid@buggregator',
                ),
            )
            ->html(
                body: <<<'TEXT'
<img src="cid:logo-embeddable">
Hello Alice.<br>This is a test message with 5 header fields and 4 lines in the message body.
TEXT
                ,
            );
    }

    public function buildEmailWithCyrillic(): Email
    {
        // Similar to buildEmail but with Cyrillic content
        return (new Email())
            ->subject('Test message with Cyrillic')
            ->date(new \DateTimeImmutable('2024-05-02 16:01:33'))
            ->addTo(
                new Address('alice@example.com', 'Alice Doe'),
                'barney@example.com',
            )
            ->addFrom(new Address('no-reply@site.com', 'Bob Example'), )
            ->attachFromPath(path: __DIR__ . '/hello.txt', )
            ->attachFromPath(path: __DIR__ . '/logo.svg')
            ->addPart(
                (new DataPart(new File(__DIR__ . '/logo.svg'), 'logo-embeddable'))->asInline()->setContentId(
                    'test-cid@buggregator',
                ),
            )
            ->html(
                body: <<<'TEXT'
<img src="cid:logo-embeddable">
<p>съешь же ещё этих мягких французских булок, да выпей чаю</p>
<p>съешь же ещё этих мягких французских булок, да выпей чаю</p>
<p>съешь же ещё этих мягких французских булок, да выпей чаю</p>
<p>съешь же ещё этих мягких французских булок, да выпей чаю</p>
<p>съешь же ещё этих мягких французских булок, да выпей чаю</p>
TEXT
                ,
            );
    }
}
