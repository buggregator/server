<?php

declare(strict_types=1);

namespace Tests\Feature\Interfaces\TCP\Smtp;

use App\Application\Commands\HandleReceivedEvent;
use Mockery\MockInterface;
use Modules\Smtp\Application\Mail\Message as MailMessage;
use Modules\Smtp\Application\Storage\EmailBodyStorage;
use Modules\Smtp\Application\Storage\Message;
use Modules\Smtp\Domain\AttachmentStorageInterface;
use Modules\Smtp\Interfaces\TCP\Service;
use Psr\SimpleCache\CacheInterface;
use Ramsey\Uuid\Uuid;
use Spiral\Cqrs\CommandBusInterface;
use Spiral\RoadRunner\Tcp\Request;
use Spiral\RoadRunner\Tcp\TcpEvent;
use Spiral\RoadRunnerBridge\Tcp\Response\RespondMessage;
use Tests\TestCase;
use Tests\Utilities\ParserTestHelper;

final class ServiceTest extends TestCase
{
    private EmailBodyStorage $emailBodyStorage;
    private MockInterface|AttachmentStorageInterface $attachments;
    private MockInterface|CommandBusInterface $bus;
    private MockInterface|CacheInterface $cache;
    private Service $service;

    protected function setUp(): void
    {
        parent::setUp();

        // Reset the ParserFactory
        ParserTestHelper::resetParserFactory();

        // Create a mock for the cache that EmailBodyStorage depends on
        $this->cache = $this->createMock(CacheInterface::class);

        // Create a real EmailBodyStorage with a mock cache
        $this->emailBodyStorage = new EmailBodyStorage($this->cache);

        $this->attachments = $this->createMock(AttachmentStorageInterface::class);
        $this->bus = $this->createMock(CommandBusInterface::class);

        $this->service = new Service(
            $this->bus,
            $this->emailBodyStorage,
            $this->attachments,
        );
    }

    protected function tearDown(): void
    {
        ParserTestHelper::resetParserFactory();
        parent::tearDown();
    }

    public function testResetBodyOnDataCommand(): void
    {
        $connectionUuid = (string) Uuid::uuid4();
        $message = new Message($connectionUuid);
        $message->body = 'Some existing content that should be cleared';

        // Setup the cache mock to return our message with existing content
        $this->cache
            ->expects($this->once())
            ->method('get')
            ->with($connectionUuid, $this->anything())
            ->willReturn($message);

        // Expect the cache to be updated with the reset message
        $this->cache
            ->expects($this->once())
            ->method('set')
            ->with(
                $connectionUuid,
                $this->callback(fn($persistedMessage) => $persistedMessage->body === '' && $persistedMessage->waitBody === true),
                $this->anything(),
            );

        // Create a request with DATA command
        $request = new Request(
            remoteAddr: '127.0.0.1',
            event: TcpEvent::Data,
            body: "DATA\r\n",
            connectionUuid: $connectionUuid,
            server: 'test-server',
        );

        // Handle the request
        $response = $this->service->handle($request);

        // Verify response is correct
        $this->assertInstanceOf(RespondMessage::class, $response);
        $this->assertStringContainsString('354', $response->getBody());
    }

    public function testResetWaitBodyAfterMessageProcessing(): void
    {
        $connectionUuid = (string) Uuid::uuid4();
        $message = new Message($connectionUuid);
        $message->waitBody = true;
        $message->body = "Test message content\r\n.\r\n"; // With EOS marker

        // Create a real MailMessage object instead of a mock
        $mailMessage = new MailMessage(
            id: 'test-message-id',
            raw: $message->getBody(),
            sender: [['email' => 'test@example.com', 'name' => 'Test Sender']],
            recipients: [],
            ccs: [],
            subject: 'Test Subject',
            htmlBody: '<p>Test HTML</p>',
            textBody: 'Test plain text',
            replyTo: [],
            allRecipients: [],
            attachments: [],
        );

        ParserTestHelper::setupParserWithPredefinedResult($mailMessage);

        // Set up the cache mock
        $this->cache
            ->expects($this->once())
            ->method('get')
            ->with($connectionUuid, $this->anything())
            ->willReturn($message);

        // Setup mocks for dispatchMessage
        $this->attachments
            ->expects($this->once())
            ->method('store')
            ->willReturn([]);

        $this->bus
            ->expects($this->once())
            ->method('dispatch')
            ->with($this->isInstanceOf(HandleReceivedEvent::class));

        // Create a request with the last part of message body
        $request = new Request(
            remoteAddr: '127.0.0.1',
            event: TcpEvent::Data,
            body: "Final line of the message\r\n.\r\n",
            connectionUuid: $connectionUuid,
            server: 'test-server',
        );

        // Handle the request
        $response = $this->service->handle($request);

        // Verify the waitBody flag was reset
        $this->assertFalse($message->waitBody, 'waitBody flag should be reset after processing a message');

        // Verify response code is 250 (message accepted)
        $this->assertInstanceOf(RespondMessage::class, $response);
        $this->assertStringContainsString('250', $response->getBody());
    }

    public function testCorrectResponseForDataWithCyrillic(): void
    {
        $connectionUuid = (string) Uuid::uuid4();
        $message = new Message($connectionUuid);
        $message->waitBody = true;

        // Setup the cache mock
        $this->cache
            ->expects($this->once())
            ->method('get')
            ->with($connectionUuid, $this->anything())
            ->willReturn($message);

        // Expect the cache to be updated
        $this->cache
            ->expects($this->once())
            ->method('set')
            ->with(
                $connectionUuid,
                $this->callback(fn($persistedMessage) => str_contains(
                    $persistedMessage->body,
                    "съешь же ещё этих мягких французских булок, да выпей чаю",
                )),
                $this->anything(),
            );

        // Create a request with Cyrillic content
        $request = new Request(
            remoteAddr: '127.0.0.1',
            event: TcpEvent::Data,
            body: "съешь же ещё этих мягких французских булок, да выпей чаю\r\n",
            connectionUuid: $connectionUuid,
            server: 'test-server',
        );

        // Handle the request
        $response = $this->service->handle($request);

        // Verify the body contains the Cyrillic content
        $this->assertStringContainsString(
            "съешь же ещё этих мягких французских булок, да выпей чаю",
            $message->body,
        );

        // Verify correct response for partial data
        $this->assertInstanceOf(RespondMessage::class, $response);
        $this->assertStringContainsString('250', $response->getBody()); // OK response
    }
}
