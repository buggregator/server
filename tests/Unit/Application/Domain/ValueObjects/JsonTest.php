<?php

declare(strict_types=1);

namespace Application\Domain\ValueObjects;

use App\Application\Domain\ValueObjects\Json;
use PHPUnit\Framework\Attributes\DataProvider;
use PHPUnit\Framework\TestCase;

final class JsonTest extends TestCase
{
    public static function jsonDataProvider(): iterable
    {
        yield 'null' => [
            'null',
            [],
        ];

        yield 'empty-string' => [
            '',
            [],
        ];

        yield 'empty' => [
            '{}',
            [],
        ];

        yield 'simple' => [
            '{"key": "value"}',
            ['key' => 'value'],
        ];
    }

    #[DataProvider('jsonDataProvider')]
    public function testDecode(string $json, mixed $expected): void
    {
        $object = Json::typecast($json);

        $this->assertEquals($expected, $object->jsonSerialize());
    }
}
