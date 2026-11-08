<?php
require __DIR__ . '/lib.php';

function outer(): void { middle(); after(); }
function middle(): void { inner(); }
function inner(): void {
    xdebug_start_trace('/tmp/mid');
    $x = App\fib(4);
    strlen("x$x");
}
function after(): string { $r = str_repeat('y', 3); xdebug_stop_trace(); return $r; }
outer();
