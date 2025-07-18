# SMTP Attachment Processing Strategy Pattern

## Overview

This implementation introduces a strategy pattern for processing different types of email attachments in the SMTP
module. The pattern was implemented to fix the issue where inline attachments without filenames (like
`cid:qr@domain.com`) were causing `FileOperationException` when the storage system couldn't determine the MIME type.

## Problem Solved

**Original Issue**: When `Symfony\Component\Mime\Part\DataPart` is attached to an email as an inline attachment without
a filename and referenced using `cid:ATTACHMENT_ID` in the email body, the system threw:

```
Spiral\Storage\Exception\FileOperationException: Unable to retrieve the mime_type for file at location: 01973a7b-4578-70ad-a65c-0195ad63ba6a/qr@domain.com
```

## Solution Architecture

### Strategy Pattern Structure

```
AttachmentProcessingStrategy (interface)
├── InlineAttachmentStrategy (handles cid: attachments)
├── RegularAttachmentStrategy (handles normal file attachments)
└── FallbackAttachmentStrategy (handles edge cases)

AttachmentProcessorFactory (creates appropriate strategy)
AttachmentProcessor (context class)
```

### Key Components

1. **AttachmentProcessingStrategy**: Interface defining the contract for attachment processing strategies
2. **InlineAttachmentStrategy**: Handles inline attachments with content-id, generates filenames from content-id
3. **RegularAttachmentStrategy**: Handles normal file attachments with existing filenames
4. **FallbackAttachmentStrategy**: Handles edge cases and unknown attachment types
5. **AttachmentProcessorFactory**: Factory that determines which strategy to use based on attachment characteristics
6. **AttachmentProcessor**: Context class that uses the selected strategy to process attachments

## How It Works

### Strategy Selection

The factory determines which strategy to use based on:

- **InlineAttachmentStrategy**: Used when `getContentId()` returns a non-null value
- **RegularAttachmentStrategy**: Used when `getContentId()` is null or disposition is 'attachment'
- **FallbackAttachmentStrategy**: Used as a last resort (always returns true for `canHandle()`)

### Filename Generation

Each strategy has its own approach to filename generation:

#### InlineAttachmentStrategy

- Uses original filename if available
- Generates filename from content-id: `qr@domain.com` → `qr_domain_com.png`
- Adds appropriate file extension based on MIME type
- Sanitizes special characters

#### RegularAttachmentStrategy

- Uses original filename if available
- Generates unique filename with MIME type extension if no filename
- Preserves more filename characters for regular attachments

#### FallbackAttachmentStrategy

- Tries original filename first
- Falls back to content-id based generation
- Last resort: generates unique filename with timestamp

## Usage Examples

### Basic Usage in Parser

```php
// Old way (problematic)
$attachments = array_map(fn($part) => new Attachment(
    $part->getFilename(), // Could be null!
    $part->getContent(),
    $part->getContentType(),
    $part->getContentId(),
), $message->getAllAttachmentParts());

// New way (strategy pattern)
$factory = new AttachmentProcessorFactory();
$attachments = [];
foreach ($message->getAllAttachmentParts() as $part) {
    $processor = $factory->createProcessor($part);
    $attachments[] = $processor->processAttachment($part);
}
```

### Custom Strategy Registration

```php
$factory = new AttachmentProcessorFactory();
$factory->registerStrategy(new MyCustomAttachmentStrategy());
```

## Testing

The implementation includes comprehensive tests:

- **Unit tests** for each strategy
- **Integration tests** for the factory and processor
- **Feature tests** for real email parsing scenarios
- **Edge case tests** for malformed attachments

### Key Test Scenarios

1. Inline attachments without filenames (the original issue)
2. Complex content-ids with special characters
3. Mixed attachment types in single email
4. Fallback scenarios for unknown attachment types
5. Error handling for malformed attachments

## Benefits

1. **Separation of Concerns**: Each strategy handles one type of attachment
2. **Extensibility**: Easy to add new attachment types without modifying existing code
3. **Maintainability**: Changes to one attachment type don't affect others
4. **Testability**: Each strategy can be unit tested independently
5. **Robustness**: Better error handling and fallback mechanisms

## Files Created/Modified

### New Files

- `Application/Mail/Strategy/AttachmentProcessingStrategy.php`
- `Application/Mail/Strategy/InlineAttachmentStrategy.php`
- `Application/Mail/Strategy/RegularAttachmentStrategy.php`
- `Application/Mail/Strategy/FallbackAttachmentStrategy.php`
- `Application/Mail/Strategy/AttachmentProcessorFactory.php`
- `Application/Mail/AttachmentProcessor.php`

### Modified Files

- `Application/Mail/Parser.php` (refactored to use strategies)
- `Application/Storage/ParserFactory.php` (updated for new Parser constructor)
- `tests/Feature/Interfaces/TCP/Smtp/EmailTest.php` (added test cases)

### Test Files

- `tests/Unit/Modules/Smtp/Application/Mail/Strategy/InlineAttachmentStrategyTest.php`
- `tests/Unit/Modules/Smtp/Application/Mail/Strategy/AttachmentProcessorFactoryTest.php`
- `tests/Feature/Modules/Smtp/Application/Mail/InlineAttachmentParsingTest.php`
- `tests/Utilities/ParserTestHelper.php`

## Backward Compatibility

The implementation maintains backward compatibility:

- Existing `Parser` usage continues to work
- No changes to public APIs
- Only internal implementation details changed

## Future Enhancements

Potential areas for extension:

- **Encrypted attachment strategy** for PGP/S-MIME attachments
- **Virus scanning strategy** for security checks
- **Content compression strategy** for large attachments
- **Thumbnail generation strategy** for image attachments
