<?php

/**
 * Sends demo dump() calls to Buggregator's VarDumper TCP port, the way
 * Symfony's ServerDumper does in a real app.
 *
 *   VAR_DUMPER_SERVER=host.docker.internal:9912 php dumps.php
 */

declare(strict_types=1);

require __DIR__ . '/vendor/autoload.php';

use Symfony\Component\VarDumper\Cloner\VarCloner;
use Symfony\Component\VarDumper\Dumper\ContextProvider\CliContextProvider;
use Symfony\Component\VarDumper\Dumper\ContextProvider\SourceContextProvider;
use Symfony\Component\VarDumper\Server\Connection;

final class Money
{
    public function __construct(
        public readonly int $amount,
        public readonly string $currency,
    ) {}
}

final class OrderLine
{
    public function __construct(
        public readonly string $sku,
        public readonly string $name,
        public readonly int $quantity,
        public readonly Money $price,
    ) {}
}

final class Order
{
    private array $lines = [];
    private string $status = 'pending_payment';

    public function __construct(
        public readonly string $number,
        public readonly int $customerId,
        private readonly \DateTimeImmutable $placedAt,
    ) {}

    public function add(OrderLine $line): self
    {
        $this->lines[] = $line;
        return $this;
    }
}

final class Invoice
{
    public function __construct(
        public readonly string $number,
        public readonly string $account,
        public readonly Money $total,
        public readonly string $status,
        public readonly \DateTimeImmutable $dueAt,
        public readonly array $attempts,
    ) {}
}

$host = getenv('VAR_DUMPER_SERVER') ?: '127.0.0.1:9912';
$connection = new Connection('tcp://' . $host, [
    'source' => new SourceContextProvider(),
    'cli' => new CliContextProvider(),
]);
$cloner = new VarCloner();

$sent = 0;
$failed = 0;
$dump = static function (mixed $value, string $project, ?string $label = null, ?string $language = null) use ($connection, $cloner, &$sent, &$failed): void {
    $context = ['project' => $project];
    if ($label !== null) {
        $context['label'] = $label;
    }
    if ($language !== null) {
        $context['language'] = $language;
    }
    if ($connection->write($cloner->cloneVar($value)->withContext($context))) {
        $sent++;
    } else {
        $failed++;
    }
};

// --- shop ---
$order = (new Order('ORD-2026-11-48213', 1042, new \DateTimeImmutable('-12 minutes')))
    ->add(new OrderLine('SKU-1042', 'Trail Runner X (size 42)', 1, new Money(8990, 'EUR')))
    ->add(new OrderLine('SKU-2210', 'Merino Running Socks', 1, new Money(1400, 'EUR')))
    ->add(new OrderLine('SKU-3307', 'Rain Shell Jacket', 1, new Money(1600, 'EUR')));
$dump($order, 'shop', 'order before payment');

$dump([
    'cart_id' => 'c7f1e2',
    'customer' => ['id' => 1042, 'email' => 'alex.morgan@example.com', 'segment' => 'returning'],
    'items' => [
        ['sku' => 'SKU-1042', 'qty' => 1, 'price' => 89.90],
        ['sku' => 'SKU-2210', 'qty' => 1, 'price' => 14.00],
        ['sku' => 'SKU-3307', 'qty' => 1, 'price' => 16.00],
    ],
    'shipping' => 'express',
    'coupon' => null,
], 'shop', 'cart');

$dump(
    "SELECT p.id, p.name, s.available\nFROM products p\nJOIN stock_levels s ON s.sku = p.sku\nWHERE p.category_id = 12\n  AND s.available < 5\nORDER BY s.available ASC;",
    'shop',
    'low stock query',
    'sql',
);
$dump('Checkout started for cart c7f1e2', 'shop');
$dump(129.90, 'shop', 'order total');
$dump(true, 'shop', 'new-checkout-flow enabled');

// --- billing ---
$dump(new Invoice(
    'INV-2026-0042',
    'acct_9012',
    new Money(4900, 'EUR'),
    'payment_failed',
    new \DateTimeImmutable('+3 days'),
    [
        ['at' => '2026-11-01T09:00:00Z', 'result' => 'declined', 'code' => 'insufficient_funds'],
        ['at' => '2026-11-04T09:00:00Z', 'result' => 'declined', 'code' => 'insufficient_funds'],
    ],
), 'billing', 'invoice');

$dump([
    'id' => 'evt_3f9a2c',
    'type' => 'charge.failed',
    'data' => ['charge' => 'ch_81d2', 'amount' => 4900, 'currency' => 'eur', 'invoice' => 'INV-2026-0042'],
], 'billing', 'webhook payload');

$dump(
    json_encode(['country' => 'DE', 'rate' => 0.19, 'tax' => 9.31, 'total' => 58.31], JSON_PRETTY_PRINT),
    'billing',
    'tax response',
    'json',
);
$dump(1287, 'billing', 'active subscriptions');

fwrite(STDOUT, sprintf("var-dump: sent %d, failed %d (server %s)\n", $sent, $failed, $host));
exit($failed > 0 ? 1 : 0);
