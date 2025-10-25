# shadcn/ui: Component Library Reference

**Source**: https://ui.shadcn.com/llms.txt

## Overview

**shadcn/ui** is an open-source collection of accessible, well-designed components built with TypeScript, Tailwind CSS, and Radix UI primitives. The project supports multiple frameworks including Next.js, Vite, Remix, Astro, and Laravel.

## Core Identity

Shadcn/ui is "a collection of beautifully-designed, accessible components and a code distribution platform" - **not an npm package**, but a registry of components you copy into your project.

## Key Principles

The project emphasizes five core design philosophies:
- **Open Code**: Transparent, modifiable component implementations (copy-paste, own the code)
- **Composition**: Building complex UIs from smaller, reusable pieces
- **Distribution**: Registry system for publishing and sharing components
- **Beautiful Defaults**: Professional styling out of the box with Tailwind CSS
- **AI-Ready**: Designed for AI assistant integration and code generation

## Component Categories

The library includes 52+ components organized into categories:

### Form & Input (16 components)
- Button, Input, Checkbox, Radio, Switch
- Select, Combobox, Date Picker, Calendar
- Form, Label, Textarea
- Slider, Toggle, Toggle Group
- Command

### Layout & Navigation (8 components)
- Accordion, Breadcrumb, Sidebar
- Tabs, Scroll Area, Separator
- Navigation Menu, Menubar

### Overlays & Dialogs (11 components)
- Dialog, Drawer, Sheet
- Popover, Tooltip, Hover Card
- Context Menu, Dropdown Menu
- Alert Dialog, Command
- Sonner (toast notifications)

### Feedback & Status (7 components)
- Alert, Toast, Progress
- Skeleton, Spinner, Badge
- Loading indicators

### Display & Media (10 components)
- Card, Table, Chart
- Avatar, Carousel, Aspect Ratio
- Pagination, Collapsible
- Resizable panels

### Miscellaneous
- Various utility components and patterns

## Framework Support & Configuration

Installation guides available for:
- Next.js (App Router & Pages Router)
- Vite
- Remix
- Astro
- Laravel
- Gatsby
- React Router
- TanStack Router
- Manual installation

### Configuration File: `components.json`

Central configuration file for CLI and installation preferences:
```json
{
  "$schema": "https://ui.shadcn.com/schema.json",
  "style": "default | new-york",
  "rsc": false,
  "tsx": true,
  "tailwind": {
    "config": "tailwind.config.js",
    "css": "src/index.css",
    "baseColor": "slate | gray | zinc | neutral | stone",
    "cssVariables": true,
    "prefix": ""
  },
  "aliases": {
    "components": "@/components",
    "utils": "@/lib/utils"
  }
}
```

## Theming & Customization

- **CSS Variables**: Theme customization through CSS custom properties
- **Color System**: Comprehensive palette with semantic color tokens
- **Dark Mode**: Framework-specific implementations (next-themes, class-based, etc.)
- **Typography**: Configurable with Tailwind's typography system
- **Spacing & Sizing**: Based on Tailwind's spacing scale

## AI Integration (MCP Server)

The MCP (Model Context Protocol) Server enables AI assistants to:
- Browse and search component registries
- Install components using natural language
- Get usage examples and documentation
- View component details and dependencies

**Compatible with**:
- Claude Code
- Cursor
- VS Code with GitHub Copilot
- Other AI development tools

### MCP Commands Available:
- `get_project_registries` - Get configured registries
- `list_items_in_registries` - List available components
- `search_items_in_registries` - Search for components
- `view_items_in_registries` - View component details
- `get_item_examples_from_registries` - Find usage examples
- `get_add_command_for_items` - Get CLI commands to add components
- `get_audit_checklist` - Verify component setup

## Advanced Capabilities

- **Monorepo Support**: Works in monorepo setups with proper path configuration
- **React 19 Compatibility**: Supports latest React features
- **Tailwind CSS v4**: Integration with newest Tailwind version
- **JavaScript Support**: Can be used without TypeScript
- **Figma Integration**: Design resources available
- **v0 Integration**: AI-generated UI from Vercel's v0

## Registry System

Allows developers to create custom component registries:
- Standardized JSON schemas
- Define structure, dependencies, and distributable code
- Share components across projects and teams
- Private or public registries

## Best Practices for AI-Assisted Development

1. **Component Discovery**: Use MCP tools to search and find appropriate components
2. **Example-First**: Always check examples before implementing
3. **Composition**: Combine smaller components to build complex UIs
4. **Customization**: Modify copied components to fit your needs
5. **Accessibility**: Components come with ARIA attributes and keyboard navigation
6. **Consistency**: Follow the configured theme and styling patterns
7. **Path Aliases**: Use configured aliases (@/components/ui/*)

## Installation Workflow

1. Initialize shadcn/ui: `npx shadcn@latest init`
2. Add components: `npx shadcn@latest add button`
3. Or use MCP: "Add a button component using shadcn"
4. Customize as needed in your project

## Common Patterns

### Form Example
```tsx
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"

<form>
  <Label htmlFor="email">Email</Label>
  <Input id="email" type="email" />
  <Button type="submit">Submit</Button>
</form>
```

### Card with Content
```tsx
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card"

<Card>
  <CardHeader>
    <CardTitle>Title</CardTitle>
  </CardHeader>
  <CardContent>
    <p>Content here</p>
  </CardContent>
</Card>
```

### Dialog Pattern
```tsx
import { Dialog, DialogTrigger, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog"

<Dialog>
  <DialogTrigger asChild>
    <Button>Open</Button>
  </DialogTrigger>
  <DialogContent>
    <DialogHeader>
      <DialogTitle>Dialog Title</DialogTitle>
    </DialogHeader>
    <p>Dialog content</p>
  </DialogContent>
</Dialog>
```

## Resources

- **Documentation**: https://ui.shadcn.com
- **LLM Reference**: https://ui.shadcn.com/llms.txt
- **GitHub**: https://github.com/shadcn/ui
- **Examples**: https://ui.shadcn.com/examples
- **Blocks**: Pre-built component combinations
- **Charts**: Built-in charting components with Recharts
- **Themes**: Browse and customize color themes

## Project-Specific Configuration (Deeb)

Our project uses:
- **Style**: `new-york` (more refined, compact design)
- **Base Color**: `neutral`
- **Icon Library**: `lucide` (lucide-react)
- **TypeScript**: Enabled (`tsx: true`)
- **CSS Variables**: Enabled for theming
- **Path Aliases**:
  - `@/components` → components directory
  - `@/lib` → lib directory
  - `@/hooks` → hooks directory

## Key Takeaways for Implementation

1. **Not a Library**: Components are copied into your project, giving you full control
2. **Composition Over Configuration**: Build complex UIs by composing simple components
3. **Accessibility First**: All components follow ARIA best practices
4. **Customizable**: Modify any aspect since you own the code
5. **AI-Friendly**: Designed to work seamlessly with AI assistants
6. **Framework Agnostic**: Works with any React-based framework
7. **Beautiful by Default**: Professional styling without extra effort

---

**Last Updated**: 2025-01-25
**Reference for**: All frontend UI component implementations in the Deeb project
