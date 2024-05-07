<?php

declare(strict_types=1);

namespace Tests\Unit\Modules\Webhooks;

use App\Application\Exception\InvalidArgumentException;
use Modules\Webhooks\Domain\ValueObject\Url;
use PHPUnit\Framework\Attributes\DataProvider;
use Tests\TestCase;

final class UrlTest extends TestCase
{
    public static function validUrlsProvider(): iterable
    {
        return [
            ['http://example.com'],
            ['https://example.com'],
            ['http://example.com/path'],
            ['https://example.com/path'],
            ['http://example.com/path?query=string'],
            ['https://example.com/path?query=string'],
        ];
    }

    #[DataProvider('validUrlsProvider')]
    public function testValidUrl(string $url): void
    {
        $this->assertSame($url, (string) Url::create($url));
        $this->assertSame(\json_encode($url), \json_encode(Url::create($url)));
    }

    public static function invalidUrlsProvider(): iterable
    {
        return [
            ['invalid-url'],
            ['ftp://example.com'],
            ['http//example.com'],
            ['http:/example.com'],
        ];
    }

    #[DataProvider('invalidUrlsProvider')]
    public function testInvalidUrl(string $url): void
    {
        $this->expectException(InvalidArgumentException::class);

        Url::create($url);
    }
}
