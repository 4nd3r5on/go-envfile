# go-envfile

A Go library for parsing and editing `.env` files while preserving formatting, comments, and style.

## Features

- **Parse .env files** into structured data
- **Update variables** while maintaining file integrity
- **Preserve formatting**: Comments, `export` keywords, quote styles, and whitespace are kept intact
- **Quote preservation**: Variables that were quoted remain quoted after updates, even without spaces
- **Section support**: Group related variables (experimental feature)

## Installation

```bash
go get github.com/4nd3r5on/go-envfile
```

## Quick Start

```go
package main

import (
    "fmt"
    "log"

    "github.com/4nd3r5on/go-envfile"
    "github.com/4nd3r5on/go-envfile/updater"
)

func main() {
    // Parse the .env file
    parsedLines, err := envfile.ParseFile("./.env", envfile.NewParser())
    if err != nil {
        log.Fatal(err)
    }

    // Convert to a map for easy access
    varMap := envfile.LinesToVariableMap(parsedLines)

    // Print all variables
    for key, value := range varMap {
        fmt.Printf("Found variable %s=%s\n", key, value)
    }

    // Update variables
    err = envfile.UpdateFile(
        "./.env",
        []updater.Update{
            {
                Key:   "DATABASE_URL",
                Value: "postgres://localhost:5432/mydb",
            },
            {
                Key:   "API_KEY",
                Value: "secret-key-here",
            },
        },
        envfile.UpdateFileOptions{
            Backup: true,  // Creates a .env.bak backup file
        },
    )
    if err != nil {
        log.Fatal(err)
    }
}
```

## API Reference

### Parsing

#### `ParseFile(filename string, parser Parser) ([]Line, error)`

Parses a .env file and returns a slice of parsed lines.

**Parameters:**
- `filename`: Path to the .env file
- `parser`: Parser instance (use `envfile.NewParser()`)

**Returns:** Slice of `Line` objects containing parsed data

#### `LinesToVariableMap(lines []Line) map[string]string`

Converts parsed lines into a simple key-value map.

**Parameters:**
- `lines`: Parsed lines from `ParseFile`

**Returns:** Map of environment variable names to values

### Updating

#### `UpdateFile(filename string, updates []updater.Update, options UpdateFileOptions) error`

Updates variables in a .env file while preserving formatting.

**Parameters:**
- `filename`: Path to the .env file
- `updates`: Slice of updates to apply
- `options`: Configuration options

**Update Fields:**
- `Key`: Variable name (required)
- `Value`: New value (required)
- `Section`: Optional section name for grouping

**UpdateFileOptions Fields:**
- `Backup`: If `true`, creates a `.bak` backup before updating
- `Logger`: Optional `*slog.Logger` for debug output

### Working with Sections

Sections allow you to group related variables:

```go
envfile.UpdateFile(
    "./.env",
    []updater.Update{
        {
            Key:     "DB_HOST",
            Value:   "localhost",
            Section: "database",
        },
        {
            Key:     "DB_PORT",
            Value:   "5432",
            Section: "database",
        },
        {
            Key:     "API_HOST",
            Value:   "api.example.com",
            Section: "api",
        },
    },
    envfile.UpdateFileOptions{},
)
```

**Note:** Section support is experimental and not fully tested yet.

## Advanced Usage

### Custom Logging

```go
import (
    "log/slog"
    "os"

    "github.com/4nd3r5on/go-envfile"
    "github.com/4nd3r5on/go-envfile/updater"
)

logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug,
}))

envfile.UpdateFile(
    "./.env",
    updates,
    envfile.UpdateFileOptions{
        Backup: true,
        Logger: logger,
    },
)
```

### Handling Spaces in Values

The library automatically handles values with spaces:

```go
envfile.UpdateFile(
    "./.env",
    []updater.Update{
        {
            Key:   "MESSAGE",
            Value: "Hello World",  // Will be properly quoted if needed
        },
    },
    envfile.UpdateFileOptions{},
)
```

## Format Preservation

The library maintains your .env file's original style:

- **Comments** are preserved in their original positions
- **`export` keywords** are kept if present
- **Quote style** (single/double) is maintained
- **Whitespace** around `=` is preserved
- **Empty lines** remain intact

### Example

**Before:**
```bash
# Database configuration
export DB_HOST="localhost"
DB_PORT=5432

# API settings
API_KEY='secret-key'
```

**After updating `DB_PORT` to `3306`:**
```bash
# Database configuration
export DB_HOST="localhost"
DB_PORT=3306

# API settings
API_KEY='secret-key'
```

## Roadmap

- [ ] Improve updater strategy for variable placement
- [ ] More testing for section features
- [ ] More unit tests coverage
