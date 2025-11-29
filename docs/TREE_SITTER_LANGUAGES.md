# Tree-Sitter Language Support

This document details the programming languages supported by the Code Indexing System, including which symbol types are extracted and any language-specific considerations.

## Supported Languages Overview

| Language | ID | Extensions | Symbol Extraction | Notes |
|----------|-----|------------|-------------------|-------|
| Go | `go` | `.go` | ✅ Full | Excellent support |
| TypeScript | `typescript` | `.ts`, `.mts`, `.cts` | ✅ Full | Including decorators |
| JavaScript | `javascript` | `.js`, `.mjs`, `.cjs`, `.jsx` | ✅ Full | ES6+ supported |
| TSX | `tsx` | `.tsx` | ✅ Full | React components |
| Python | `python` | `.py`, `.pyw`, `.pyi` | ✅ Full | Classes, functions, decorators |
| Java | `java` | `.java` | ✅ Full | Full OOP support |
| Kotlin | `kotlin` | `.kt`, `.kts` | ✅ Full | Including data classes |
| Rust | `rust` | `.rs` | ✅ Full | Traits, impls, macros |
| PHP | `php` | `.php`, `.phtml` | ✅ Full | Classes, traits, interfaces |
| Swift | `swift` | `.swift` | ✅ Full | Protocols, extensions |
| C | `c` | `.c` | ✅ Full | Functions, structs |
| C++ | `cpp` | `.cpp`, `.cc`, `.cxx`, `.hpp` | ✅ Full | Classes, templates |
| C# | `csharp` | `.cs` | ✅ Full | Full .NET support |
| Ruby | `ruby` | `.rb`, `.rake`, `.gemspec` | ✅ Full | Modules, classes |
| Scala | `scala` | `.scala`, `.sc` | ✅ Full | Objects, traits |
| Bash | `bash` | `.sh`, `.bash`, `.zsh` | ⚠️ Partial | Functions only |

## Language Details

### Go

**Extensions**: `.go`

**Extracted Symbols**:
- `package` - Package declarations
- `function` - Standalone functions
- `method` - Methods with receivers
- `struct` - Struct type definitions
- `interface` - Interface definitions
- `type` - Type aliases
- `constant` - Const declarations
- `variable` - Var declarations

**Example**:
```go
package main

type UserService struct {
    db *Database
}

func (s *UserService) GetUser(id string) (*User, error) {
    return s.db.Find(id)
}

func NewUserService(db *Database) *UserService {
    return &UserService{db: db}
}
```

**Symbol Paths**:
- `UserService` - The struct
- `UserService/GetUser` - The method
- `NewUserService` - Standalone function

---

### TypeScript

**Extensions**: `.ts`, `.mts`, `.cts`

**Extracted Symbols**:
- `class` - Class declarations
- `interface` - Interface definitions
- `function` - Functions and arrow functions
- `method` - Class methods
- `property` - Class properties
- `type` - Type aliases
- `enum` - Enum declarations
- `variable` - Const/let/var declarations

**Example**:
```typescript
interface User {
    id: string;
    name: string;
}

class UserService {
    private users: Map<string, User>;

    constructor() {
        this.users = new Map();
    }

    async getUser(id: string): Promise<User | undefined> {
        return this.users.get(id);
    }
}
```

**Symbol Paths**:
- `User` - The interface
- `UserService` - The class
- `UserService/constructor` - Constructor
- `UserService/getUser` - Method

---

### JavaScript

**Extensions**: `.js`, `.mjs`, `.cjs`, `.jsx`

**Extracted Symbols**:
- `class` - ES6 classes
- `function` - Function declarations
- `method` - Class methods
- `variable` - Const/let/var with functions

**Notes**:
- JSX/React components are supported
- CommonJS and ES modules both work
- Arrow functions assigned to variables are captured

---

### TSX

**Extensions**: `.tsx`

Same as TypeScript, with additional support for:
- React functional components
- JSX elements (not extracted as symbols)

---

### Python

**Extensions**: `.py`, `.pyw`, `.pyi`

**Extracted Symbols**:
- `class` - Class definitions
- `function` - Functions
- `method` - Class methods (including `__init__`)
- `property` - Properties with `@property`
- `variable` - Module-level assignments

**Example**:
```python
class UserService:
    def __init__(self, db):
        self.db = db

    def get_user(self, user_id: str) -> Optional[User]:
        return self.db.find(user_id)

    @property
    def connection(self):
        return self.db.connection
```

**Symbol Paths**:
- `UserService` - The class
- `UserService/__init__` - Constructor
- `UserService/get_user` - Method
- `UserService/connection` - Property

---

### Java

**Extensions**: `.java`

**Extracted Symbols**:
- `class` - Class declarations
- `interface` - Interface definitions
- `enum` - Enum types
- `method` - Methods
- `field` - Class fields
- `constructor` - Constructors

**Example**:
```java
public class UserService {
    private final Database db;

    public UserService(Database db) {
        this.db = db;
    }

    public User getUser(String id) {
        return db.find(id);
    }
}
```

---

### Kotlin

**Extensions**: `.kt`, `.kts`

**Extracted Symbols**:
- `class` - Regular and data classes
- `object` - Singleton objects
- `interface` - Interface definitions
- `function` - Top-level and extension functions
- `method` - Class methods
- `property` - Properties (val/var)

**Notes**:
- Data classes are fully supported
- Companion objects are extracted
- Extension functions maintain their receiver type context

---

### Rust

**Extensions**: `.rs`

**Extracted Symbols**:
- `struct` - Struct definitions
- `enum` - Enum types
- `trait` - Trait definitions
- `impl` - Implementation blocks
- `function` - Functions
- `method` - Impl methods
- `constant` - Const items
- `type` - Type aliases

**Example**:
```rust
pub struct UserService {
    db: Database,
}

impl UserService {
    pub fn new(db: Database) -> Self {
        Self { db }
    }

    pub fn get_user(&self, id: &str) -> Option<User> {
        self.db.find(id)
    }
}
```

---

### PHP

**Extensions**: `.php`, `.phtml`, `.php3`, `.php4`, `.php5`, `.phps`

**Extracted Symbols**:
- `class` - Class declarations
- `interface` - Interface definitions
- `trait` - Trait definitions
- `function` - Functions
- `method` - Class methods
- `property` - Class properties

---

### Swift

**Extensions**: `.swift`

**Extracted Symbols**:
- `class` - Class definitions
- `struct` - Struct definitions
- `protocol` - Protocol definitions
- `enum` - Enum types
- `extension` - Extensions
- `function` - Functions
- `method` - Instance methods
- `property` - Properties

---

### C

**Extensions**: `.c`

**Extracted Symbols**:
- `function` - Function definitions
- `struct` - Struct definitions
- `enum` - Enum types
- `typedef` - Type definitions
- `variable` - Global variables

**Notes**:
- Header files (`.h`) are associated with C by default
- Macros are not extracted as symbols

---

### C++

**Extensions**: `.cpp`, `.cc`, `.cxx`, `.hpp`, `.hxx`, `.hh`

**Extracted Symbols**:
- `class` - Class definitions
- `struct` - Struct definitions
- `function` - Functions
- `method` - Class methods
- `namespace` - Namespace definitions
- `enum` - Enum types
- `template` - Template definitions

**Notes**:
- Templates are captured with their parameters
- Namespaces create hierarchical symbol paths

---

### C#

**Extensions**: `.cs`

**Extracted Symbols**:
- `class` - Class definitions
- `interface` - Interface definitions
- `struct` - Struct definitions
- `enum` - Enum types
- `method` - Methods
- `property` - Properties
- `field` - Fields
- `namespace` - Namespace definitions

---

### Ruby

**Extensions**: `.rb`, `.rake`, `.gemspec`

**Extracted Symbols**:
- `class` - Class definitions
- `module` - Module definitions
- `method` - Methods (def)
- `singleton_method` - Class methods

**Notes**:
- Blocks are not extracted as separate symbols
- Dynamic method definitions (define_method) are not captured

---

### Scala

**Extensions**: `.scala`, `.sc`

**Extracted Symbols**:
- `class` - Class definitions
- `object` - Object definitions
- `trait` - Trait definitions
- `def` - Method definitions
- `val` - Value definitions
- `var` - Variable definitions
- `type` - Type aliases

---

### Bash

**Extensions**: `.sh`, `.bash`, `.zsh`

**Extracted Symbols**:
- `function` - Function definitions

**Notes**:
- Only function definitions are extracted
- Variables and aliases are not captured
- Limited symbol hierarchy support

---

## Adding Language Support

The code indexing system uses [go-tree-sitter](https://github.com/smacker/go-tree-sitter) for parsing. To add a new language:

1. **Check availability**: Ensure a tree-sitter grammar exists for the language
2. **Add to languages.go**: Register the language with its extensions and grammar
3. **Create extractor**: Implement symbol extraction in `pkg/treesitter/extractors/`
4. **Test**: Add tests for the new language

### Language Registration

In `pkg/treesitter/languages.go`:

```go
var supportedLanguages = map[Language]LanguageInfo{
    LanguageNewLang: {
        Language:   LanguageNewLang,
        Name:       "New Language",
        Extensions: []string{"ext1", "ext2"},
        Grammar:    newlang.GetLanguage,
    },
}
```

### Symbol Extraction

Each language needs an extractor that walks the AST and identifies symbols. See existing extractors in `pkg/treesitter/extractors/` for examples.

## Performance Considerations

### Parsing Speed by Language

| Language | Relative Speed | Notes |
|----------|---------------|-------|
| Go | ⚡ Fast | Simple grammar |
| JavaScript | ⚡ Fast | Well-optimized grammar |
| TypeScript | 🔶 Medium | Larger grammar due to types |
| Python | ⚡ Fast | Simple grammar |
| Java | 🔶 Medium | Complex grammar |
| C++ | 🔴 Slower | Very complex grammar |
| Rust | 🔶 Medium | Macro complexity |

### Memory Usage

Languages with more complex grammars (C++, TypeScript) may use more memory during parsing. The system handles this efficiently by:

1. Processing files one at a time per worker
2. Releasing parser memory after each file
3. Caching parser instances per language

## See Also

- [CODE_INDEXING.md](CODE_INDEXING.md) - User guide
- [CODE_INDEXING_API.md](CODE_INDEXING_API.md) - API reference
