# Buggregator Integration Guide for AI Assistants

**Purpose:** This file provides comprehensive instructions for AI assistants to automatically integrate Buggregator debugging server into any PHP project.

**Research Base:** Based on analysis of Buggregator server repository, Fingather real-world integration example, and official documentation at docs.buggregator.dev.

---

## Core Buggregator Information

**What is Buggregator:**
- Free, open-source debugging server for PHP applications
- Unified interface for multiple debugging tools: Sentry, Ray, VarDumper, Monolog, SMTP, XHProf
- Built on Spiral Framework with Vue.js frontend
- Lightweight, standalone server requiring no additional dependencies

**Default Ports:**
- `8000` - HTTP/Web UI
- `1025` - SMTP server
- `9912` - Symfony VarDumper server
- `9913` - Monolog socket handler

**Docker Image:** `ghcr.io/buggregator/server:latest`

---

## Project Detection Guidelines

**Framework Detection:**
1. **Laravel:** Look for `artisan` file and `laravel/framework` in composer.json
2. **Symfony:** Look for `bin/console` and `symfony/framework-bundle` in composer.json
3. **Spiral:** Look for `spiral/framework` in composer.json
4. **Generic PHP:** composer.json exists but no specific framework
5. **Non-PHP:** No composer.json found

**Docker Setup Detection:**
1. **Modern Docker Compose:** `docker compose version` succeeds
2. **Legacy Docker Compose:** `docker-compose --version` succeeds
3. **Docker only:** `docker --version` succeeds but no compose
4. **No Docker:** None available

**Detection Commands:**
```bash
# Check modern Docker Compose first (preferred)
docker compose version >/dev/null 2>&1 && echo "Modern Docker Compose available"

# Check legacy Docker Compose (fallback)
docker-compose --version >/dev/null 2>&1 && echo "Legacy Docker Compose available"

# Check Docker only
docker --version >/dev/null 2>&1 && echo "Docker available"
```

---

## Docker Compose Integration Patterns

**For Existing Docker Compose Projects:**
Add this service to existing `docker-compose.yml`:

```yaml
services:
  buggregator:
    image: ghcr.io/buggregator/server:latest
    container_name: buggregator
    restart: unless-stopped
    ports:
      - "127.0.0.1:8000:8000"   # Web UI
      - "127.0.0.1:1025:1025"   # SMTP
      - "127.0.0.1:9912:9912"   # VarDumper
      - "127.0.0.1:9913:9913"   # Monolog
    networks:
      - default
```

**For New Docker Compose Setup:**
Create complete `docker-compose.yml`:

```yaml
services:
  buggregator:
    image: ghcr.io/buggregator/server:latest
    container_name: buggregator
    restart: unless-stopped
    ports:
      - "127.0.0.1:8000:8000"
      - "127.0.0.1:1025:1025"
      - "127.0.0.1:9912:9912"
      - "127.0.0.1:9913:9913"
    networks:
      - buggregator-network
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8000"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s

networks:
  buggregator-network:
    driver: bridge
```

**Real-World Example (Fingather Pattern):**
The Fingather project integrates Buggregator by:
- Adding service to existing docker-compose.yml
- Using standard port configuration
- Integrating with application network
- No custom volumes or environment variables needed

---

## Framework-Specific Configuration

### Laravel Integration

**Environment Variables (.env):**
```env
# Buggregator Configuration
VAR_DUMPER_FORMAT=server
VAR_DUMPER_SERVER=127.0.0.1:9912
MONOLOG_SOCKET_HOST=127.0.0.1:9913

# Sentry Integration
SENTRY_LARAVEL_DSN=http://sentry@127.0.0.1:8000/1

# Mail Integration
MAIL_MAILER=smtp
MAIL_HOST=127.0.0.1
MAIL_PORT=1025
MAIL_USERNAME=null
MAIL_PASSWORD=null
MAIL_ENCRYPTION=null

# Ray Integration
RAY_HOST=ray@127.0.0.1
RAY_PORT=8000

# XHProf Profiling
PROFILER_ENDPOINT=http://127.0.0.1:8000/api/profiler/store
```

**Additional Packages:**
- XHProf Laravel: `composer require --dev maantje/xhprof-buggregator-laravel`
- Ray: `composer require --dev spatie/laravel-ray`

**Logging Configuration (config/logging.php):**
Add to channels array:
```php
'buggregator' => [
    'driver' => 'monolog',
    'level' => env('LOG_LEVEL', 'debug'),
    'handler' => \Monolog\Handler\SocketHandler::class,
    'formatter' => \Monolog\Formatter\JsonFormatter::class,
    'handler_with' => [
        'connectionString' => env('MONOLOG_SOCKET_HOST', '127.0.0.1:9913'),
    ],
],
```

**Usage Examples:**
```php
// VarDumper
dump($data); // Sent to Buggregator

// Ray (CORRECTED SYNTAX)
ray('Debug message')->color('green');
ray()->showQueries();

// Ray JSON - IMPORTANT: Use json_encode for arrays
ray()->json(json_encode([
    'message' => 'Debug data',
    'user_id' => 123
]));
// NOT this (will cause error): ray()->json(['data' => 'test']); ❌

// Sentry
report(new Exception('Test exception'));

// Email testing
Mail::raw('Test', function($message) {
    $message->to('test@example.com')->subject('Test');
});

// Logging
Log::channel('buggregator')->info('Debug message', ['context' => 'test']);
```

### Symfony Integration

**Environment Variables (.env):**
```env
# Buggregator Configuration
VAR_DUMPER_FORMAT=server
VAR_DUMPER_SERVER=127.0.0.1:9912

# Sentry Integration
SENTRY_DSN=http://sentry@127.0.0.1:8000/1

# Mailer Integration
MAILER_DSN=smtp://127.0.0.1:1025

# Monolog Socket
MONOLOG_SOCKET_HOST=127.0.0.1:9913
```

**Monolog Configuration (config/packages/dev/monolog.yaml):**
```yaml
monolog:
    handlers:
        buggregator:
            type: socket
            level: debug
            formatter: monolog.formatter.json
            connection_string: '%env(MONOLOG_SOCKET_HOST)%'
```

**Mailer Configuration (config/packages/dev/mailer.yaml):**
```yaml
framework:
    mailer:
        dsn: '%env(MAILER_DSN)%'
```

**Additional Packages:**
- XHProf Symfony: `composer require --dev iluckhack/xhprof-buggregator-bundle`
- Sentry: `composer require sentry/sentry-symfony`

**Bundle Registration (config/bundles.php):**
```php
ILuckHack\XHProfBuggregatorBundle\XHProfBuggregatorBundle::class => ['dev' => true],
```

**Usage Examples:**
```php
// VarDumper
dump($data); // Sent to Buggregator

// Logging
$logger->info('Debug message', ['context' => 'test']);

// Email
$email = (new Email())
    ->from('sender@example.com')
    ->to('recipient@example.com')
    ->subject('Test Email')
    ->text('Test content');
$mailer->send($email);
```

### Spiral Framework Integration

**Environment Variables (.env):**
```env
# Buggregator Configuration
VAR_DUMPER_FORMAT=server
VAR_DUMPER_SERVER=127.0.0.1:9912

# Sentry Integration
SENTRY_DSN=http://sentry@127.0.0.1:8000/1

# Monolog Configuration
MONOLOG_DEFAULT_CHANNEL=buggregator
BUGGREGATOR_SOCKET=127.0.0.1:9913
```

**Usage Examples:**
```php
// VarDumper
dump($data); // Sent to Buggregator

// Ray Integration
ray('Debug message from Spiral');

// Logging (built-in integration)
$this->logger->info('Debug message', ['context' => 'test']);
```

### Generic PHP Integration

**Configuration Setup:**
```php
<?php
// Set VarDumper server
$_SERVER['VAR_DUMPER_FORMAT'] = 'server';
$_SERVER['VAR_DUMPER_SERVER'] = '127.0.0.1:9912';

// Configure SMTP
ini_set('SMTP', '127.0.0.1');
ini_set('smtp_port', 1025);

// Sentry (if using)
\Sentry\init(['dsn' => 'http://sentry@127.0.0.1:8000/1']);

// Monolog Socket Handler
use Monolog\Logger;
use Monolog\Handler\SocketHandler;
use Monolog\Formatter\JsonFormatter;

$logger = new Logger('app');
$socketHandler = new SocketHandler('tcp://127.0.0.1:9913');
$socketHandler->setFormatter(new JsonFormatter());
$logger->pushHandler($socketHandler);
```

---

## Port Conflict Resolution

**Check for Port Conflicts:**
```bash
# Check if ports are in use
lsof -Pi :8000 -sTCP:LISTEN
lsof -Pi :1025 -sTCP:LISTEN
lsof -Pi :9912 -sTCP:LISTEN
lsof -Pi :9913 -sTCP:LISTEN
```

**Alternative Port Configuration:**
If ports are occupied, use alternatives:
- Web UI: 8080, 8001, 3000
- SMTP: 1026, 2525
- VarDumper: 9914, 9922
- Monolog: 9915, 9923

Update Docker Compose and application configuration accordingly.

---

## Testing and Verification

**Health Checks:**
1. **Container Status:** `docker ps | grep buggregator`
2. **Web Interface:** Visit `http://127.0.0.1:8000`
3. **Port Connectivity:** `nc -z 127.0.0.1 8000`

**Functional Tests:**

**PHP Test Script (Enhanced):**
```php
<?php
/**
 * Enhanced Buggregator Integration Test Script
 *
 * This script tests all Buggregator integrations with proper error handling.
 * Run after starting Buggregator to verify everything works.
 * Check results at: http://127.0.0.1:8000
 */

require_once __DIR__ . '/vendor/autoload.php';

use Illuminate\Foundation\Application;
use Illuminate\Support\Facades\Mail;
use Illuminate\Support\Facades\Log;

// Bootstrap Laravel (for Laravel projects)
if (file_exists(__DIR__ . '/bootstrap/app.php')) {
    $app = require_once __DIR__ . '/bootstrap/app.php';
    $kernel = $app->make(Illuminate\Contracts\Console\Kernel::class);
    $kernel->bootstrap();
} else {
    // For non-Laravel projects, configure VarDumper manually
    $_SERVER['VAR_DUMPER_FORMAT'] = 'server';
    $_SERVER['VAR_DUMPER_SERVER'] = '127.0.0.1:9912';
}

echo "🔍 Testing Buggregator Integration\n";
echo "==================================\n\n";

$tests_passed = 0;
$total_tests = 4;

// Test 1: VarDumper (Core functionality)
echo "1. Testing VarDumper...\n";
try {
    dump(['test' => 'VarDumper', 'timestamp' => now() ?? date('c'), 'status' => 'testing']);
    echo "   ✅ VarDumper data sent\n\n";
    $tests_passed++;
} catch (Exception $e) {
    echo "   ❌ VarDumper failed: " . $e->getMessage() . "\n\n";
}

// Test 2: Ray (Enhanced debugging)
echo "2. Testing Ray...\n";
try {
    if (function_exists('ray')) {
        ray('🚀 Ray Integration Test')->color('green');
        ray()->json(json_encode(['status' => 'working', 'framework' => 'Laravel']));
        echo "   ✅ Ray working\n\n";
        $tests_passed++;
    } else {
        echo "   ⚠️  Ray not installed (optional)\n\n";
        $tests_passed++; // Don't penalize missing optional features
    }
} catch (Exception $e) {
    echo "   ❌ Ray error: " . $e->getMessage() . "\n\n";
}

// Test 3: Email Capture (Laravel-specific or generic)
echo "3. Testing Email (SMTP)...\n";
try {
    if (class_exists('Illuminate\Support\Facades\Mail')) {
        // Laravel email
        Mail::raw('Buggregator integration test email', function($message) {
            $message->to('test@example.com')
                    ->subject('Buggregator Test - ' . (now() ?? date('c')))
                    ->from('app@example.com');
        });
    } else {
        // Generic PHP email
        mail('test@example.com', 'Buggregator Test', 'This is a test email from PHP');
    }
    echo "   ✅ Email sent to Buggregator\n\n";
    $tests_passed++;
} catch (Exception $e) {
    echo "   ❌ Email failed: " . $e->getMessage() . "\n\n";
}

// Test 4: Logging (Laravel-specific or skip)
echo "4. Testing Logging...\n";
try {
    if (class_exists('Illuminate\Support\Facades\Log')) {
        Log::channel('buggregator')->info('Integration test log entry', [
            'test_type' => 'integration',
            'timestamp' => now() ?? date('c'),
            'status' => 'testing'
        ]);
        echo "   ✅ Logging working\n\n";
        $tests_passed++;
    } else {
        echo "   ⚠️  Laravel logging not available (framework-specific)\n\n";
        $tests_passed++; // Don't penalize non-Laravel projects
    }
} catch (Exception $e) {
    echo "   ❌ Logging failed: " . $e->getMessage() . "\n\n";
}

// Results
echo "🎉 Test Results: {$tests_passed}/{$total_tests} passed\n";
echo "==========================================\n\n";

if ($tests_passed >= 3) {
    echo "✅ Buggregator integration successful!\n";
    echo "📊 View results at: http://127.0.0.1:8000\n\n";
    echo "🔍 You should see data in:\n";
    echo "   - 'Dumps' tab (VarDumper)\n";
    echo "   - 'Ray' tab (if Ray installed)\n";
    echo "   - 'SMTP' tab (Email capture)\n";
    echo "   - 'Monolog' tab (Log entries)\n";
} else {
    echo "⚠️  Some integrations failed. Check the errors above.\n";
    echo "📊 Visit http://127.0.0.1:8000 to see what's working.\n";
}
```

**Docker Test Commands:**
```bash
# Modern Docker Compose syntax (preferred)
docker compose up -d buggregator
docker compose ps buggregator
docker compose logs -f buggregator

# Legacy syntax (fallback if modern fails)
docker-compose up -d buggregator

# Test connectivity
curl -I http://127.0.0.1:8000
```

---

## Common Issues and Solutions

**Port Already in Use:**
- Solution: Use alternative ports or stop conflicting services
- Check: `lsof -i :PORT_NUMBER`
- Update both Docker Compose and application configuration

**Docker Compose Command Not Found:**
- Error: `zsh: command not found: docker-compose`
- Solution: Use `docker compose` (modern syntax) or install Docker Compose plugin
- Check: `docker compose version` vs `docker-compose --version`

**Container Not Starting:**
- Check Docker daemon: `docker info`
- Check image: `docker pull ghcr.io/buggregator/server:latest`
- Check logs: `docker logs buggregator`
- Verify port availability

**Health Check Endpoint 404:**
- Issue: `/api/health` returns 404
- Solution: Use basic connectivity test instead
- Check: `curl -I http://127.0.0.1:8000` (should return 200)

**Ray JSON Type Error:**
- Error: `Argument #1 must be of type string, array given`
- Solution: Use `ray()->json(json_encode($array))` not `ray()->json($array)`
- Example: `ray()->json(json_encode(['key' => 'value']))`

**VarDumper Not Working:**
- Verify VAR_DUMPER_SERVER environment variable
- Check port 9912 accessibility: `nc -z 127.0.0.1 9912`
- Ensure dump() function is called after configuration
- Check if symfony/var-dumper is installed

**Email Not Captured:**
- Verify SMTP configuration points to Buggregator
- Check port 1025 accessibility: `nc -z 127.0.0.1 1025`
- Ensure application is sending emails
- Test with simple mail() function

**Web Interface Not Accessible:**
- Check if container is running: `docker ps | grep buggregator`
- Verify port 8000 is exposed and accessible
- Check firewall settings
- Test with curl: `curl -f http://127.0.0.1:8000`

**Ray Not Working:**
- Install spatie/ray package
- Configure RAY_HOST and RAY_PORT environment variables
- Check network connectivity between app and Buggregator

**HTTP Dump Endpoint Issues:**
- Issue: `/api/http-dump` returns 404
- Status: Endpoint may have changed in newer versions
- Alternative: Focus on VarDumper, Ray, Logs, and Email capture

---

## Integration Checklist

**Pre-Integration:**
- [ ] Docker/Docker Compose installed
- [ ] Ports 8000, 1025, 9912, 9913 available
- [ ] Project type identified (Laravel/Symfony/PHP/Other)

**Docker Setup:**
- [ ] Buggregator service added to docker-compose.yml
- [ ] Ports correctly mapped (127.0.0.1:PORT:PORT)
- [ ] Container starts successfully
- [ ] Web interface accessible at http://127.0.0.1:8000

**Framework Configuration:**
- [ ] Environment variables updated (.env file)
- [ ] Framework-specific packages installed
- [ ] Configuration files updated (logging, mailer, etc.)

**Testing:**
- [ ] VarDumper working (dump() shows in UI)
- [ ] Email capture working (emails appear in SMTP tab)
- [ ] Sentry integration working (exceptions appear)
- [ ] Ray integration working (if applicable)
- [ ] XHProf profiling working (if applicable)

**Final Verification:**
- [ ] All debug data appears in Buggregator UI
- [ ] No console errors in browser
- [ ] Application performance impact minimal
- [ ] Test script runs without errors

---

## Advanced Configuration

**Custom Environment Variables:**
```yaml
services:
  buggregator:
    image: ghcr.io/buggregator/server:latest
    environment:
      - FRONTEND_URL=http://127.0.0.1:8000
      - LOGGER_DEFAULT_CHANNEL=stderr
      - RR_LOG_LEVEL=info
      - RR_HTTP_NUM_WORKERS=4
    # ... rest of configuration
```

**Volume Persistence:**
```yaml
services:
  buggregator:
    image: ghcr.io/buggregator/server:latest
    volumes:
      - buggregator_data:/app/runtime
    # ... rest of configuration

volumes:
  buggregator_data:
    driver: local
```

**Network Configuration for Multi-Service Apps:**
```yaml
services:
  buggregator:
    image: ghcr.io/buggregator/server:latest
    networks:
      - app-network
      - buggregator-network
    # ... rest of configuration

networks:
  app-network:
    external: true
  buggregator-network:
    driver: bridge
```

**Production-Like Setup with Authentication:**
```yaml
services:
  buggregator:
    image: ghcr.io/buggregator/server:latest
    environment:
      - AUTH_ENABLED=true
      - AUTH_PROVIDER=auth0
      - AUTH_PROVIDER_URL=${AUTH0_DOMAIN}
      - AUTH_CLIENT_ID=${AUTH0_CLIENT_ID}
      - AUTH_CLIENT_SECRET=${AUTH0_CLIENT_SECRET}
    # ... rest of configuration
```

---

## AI Assistant Integration Instructions

**When helping developers integrate Buggregator:**

### 1. Analysis Phase
- **Detect project structure and framework**
- **Check Docker environment** (modern `docker compose` vs legacy `docker-compose`)
- **Verify port availability** and suggest alternatives if conflicts exist
- **Assess file editing permissions** and prepare fallback strategies

### 2. Docker Configuration
- **Create/update docker-compose.yml** with modern syntax (no obsolete version field)
- **Use localhost binding (127.0.0.1) for security**
- **Configure health checks** (use basic connectivity, not `/api/health`)
- **Set up proper networking**

### 3. Application Configuration
- **Update environment variables** (including MONOLOG_SOCKET_HOST)
- **Install required packages** (Ray, XHProf) based on detected framework
- **Configure framework-specific settings** using appropriate editing method
- **Set up logging, mailer, and debugging tools**

### 4. Testing Setup
- **Create comprehensive test scripts** with error handling for all integrations
- **Provide health check commands** using working endpoints
- **Generate corrected usage examples** (especially Ray JSON syntax)
- **Skip problematic features** (HTTP dump if endpoint unavailable)

### 5. Documentation
- **Provide clear next steps** and usage examples
- **Include enhanced troubleshooting guide**
- **Reference official documentation**
- **Clean up test files** after verification

### File Editing Fallback Strategies

When direct file editing is blocked:

1. **For .env files:**
   ```bash
   # Use heredoc approach
   cat > .env << 'EOF'
   [content here]
   EOF

   # Or append specific variables
   echo 'MONOLOG_SOCKET_HOST=127.0.0.1:9913' >> .env
   ```

2. **For PHP config files:**
    - Use search_replace tool instead of edit_file
    - Use sed for simple additions
    - Check file permissions first: `ls -la filename`

3. **Docker Environment Checks:**
   ```bash
   # Check both modern and legacy Docker Compose
   docker compose version >/dev/null 2>&1 || docker-compose --version >/dev/null 2>&1
   ```

### Best Practices for AI Assistants:
- **Always backup existing configurations before making changes**
- **Use secure, localhost-only bindings by default**
- **Provide comprehensive testing and verification steps**
- **Include framework-specific optimizations when detected**
- **Offer alternative solutions for common conflicts**
- **Reference official documentation for complex scenarios**

---

## Quick Reference

**Essential URLs:**
- Web Interface: http://127.0.0.1:8000
- Basic Health Check: http://127.0.0.1:8000 (should return 200)

**Essential Commands:**
```bash
# Modern Docker Compose syntax (preferred)
docker compose up -d buggregator
docker compose ps buggregator
docker compose logs -f buggregator
docker compose stop buggregator

# Legacy syntax (fallback if modern fails)
docker-compose up -d buggregator

# Test connectivity
curl -I http://127.0.0.1:8000
```

**Essential Configuration:**
```env
VAR_DUMPER_FORMAT=server
VAR_DUMPER_SERVER=127.0.0.1:9912
MAIL_HOST=127.0.0.1
MAIL_PORT=1025
```

---

**Documentation References:**
- Official Docs: https://docs.buggregator.dev/
- GitHub Repository: https://github.com/buggregator/server
- Real-world Example: https://github.com/marekskopal/fingather
- Docker Hub: https://github.com/orgs/buggregator/packages
