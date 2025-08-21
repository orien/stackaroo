# 5. CLI framework

Date: 2025-08-21

## Status

Accepted

## Context

We need to choose a CLI framework for Stackaroo that can handle multiple subcommands with various flags and options. The tool will require commands such as `deploy`, `delete`, `status`, `validate`, `list`, and `init`, each with their own specific parameters and help documentation.

Key requirements:
- Support for nested subcommands and subcommand groups
- Automatic help generation and documentation
- Flag parsing with various data types
- Shell completion support
- Good error handling and user experience
- Integration with Go ecosystem standards

The main candidates considered were:
- **Cobra**: Most popular Go CLI framework, used by kubectl, helm, hugo
- **Urfave CLI**: Simpler framework, good for basic use cases
- **Flag (stdlib)**: No dependencies but manual subcommand implementation required
- **Kong**: Struct-based approach, good type safety but less mature

## Decision

We will use **Cobra** as the CLI framework for Stackaroo.

Key factors in this decision:
- Excellent subcommand support with nested command structures
- Automatic help generation and shell completion
- Widely adopted in the Go ecosystem (kubectl, helm, docker CLI, etc.)
- Strong integration with pflag for advanced flag handling
- Good documentation and large community
- Natural fit for infrastructure tools requiring complex command hierarchies
- Integrates well with Viper for configuration management

## Consequences

**Positive:**
- Robust subcommand architecture suitable for complex CLI tools
- Automatic generation of help documentation and man pages
- Shell completion support out of the box
- Familiar patterns for users of other Go-based infrastructure tools
- Strong community support and extensive documentation
- Easy to extend with new commands and flags

**Negative:**
- Additional dependency and framework complexity
- More boilerplate code compared to simpler frameworks
- Learning curve for contributors unfamiliar with Cobra
- Potentially overkill for very simple use cases

We accept these trade-offs in favour of a professional CLI experience that scales well with feature growth and provides consistency with other infrastructure tooling.
