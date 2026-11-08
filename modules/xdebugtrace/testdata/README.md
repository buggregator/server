# Recorded Xdebug traces

Recorded on PHP 8.3.36 + Xdebug 3.4.5 from the scripts in `app/`
(this Xdebug build has no zlib, so the files were written plain and gzipped after).

```sh
php -d xdebug.mode=trace -d xdebug.start_with_request=yes -d xdebug.trace_format=1 \
    -d xdebug.collect_return=1 -d xdebug.collect_assignments=1 \
    -d xdebug.output_dir=/tmp -d xdebug.trace_output_name=main app/main.php
php -d xdebug.mode=trace -d xdebug.start_with_request=trigger -d xdebug.trace_format=1 \
    -d xdebug.collect_return=1 -d xdebug.collect_assignments=1 app/mid.php
gzip -n -9 main.xt mid.xt
```

- `main.xt.gz`: `{main}`, `require`, methods, recursion, a closure, an exception,
  strings with tab/newline (Xdebug writes them as `\t`/`\n`).
- `mid.xt.gz`: `xdebug_start_trace()` called at depth 4: the first record is at
  level 5, exits of `inner`/`middle` have no entries, `after()` follows at level 3,
  and the trace stops inside `xdebug_stop_trace()`.
